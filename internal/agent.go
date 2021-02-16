// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/app"
	"github.com/sqreen/go-agent/internal/backend"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	http_protection_types "github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/rule"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"github.com/sqreen/go-agent/internal/sqlib/sqsanitize"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
	"github.com/sqreen/go-agent/internal/version"
	"github.com/sqreen/go-libsqreen/waf"
	"golang.org/x/xerrors"
)

func Start() {
	agentInstance.start()
}

var agentInstance agentInstanceType

// agent instance holder type with synchronization
type agentInstanceType struct {
	// The agent goroutine must be started once.
	// It will asynchronously set the instance pointer.
	startOnce sync.Once
	// Instance pointer access R/W lock.
	instanceAccessLock sync.RWMutex
	instance           *AgentType
}

func (instance *agentInstanceType) get() *AgentType {
	instance.instanceAccessLock.RLock()
	defer instance.instanceAccessLock.RUnlock()
	return instance.instance
}

func (instance *agentInstanceType) set(agent *AgentType) {
	instance.instanceAccessLock.Lock()
	defer instance.instanceAccessLock.Unlock()
	instance.instance = agent
}

// Start the agent when enabled and back-off restart it when unhandled errors
// or panics occur.
//
// The algorithm is based on multiple levels of try/catch equivalents called
// here "safe calls":
// - Level 1: a safe goroutine loop retrying the agent in case of unhandled
//   error or panic.
// - Level 2: a safe call to the agent initialization.
// - Level 3: a safe call to the agent main loop.
// - Level 4, implicit here: internal agent errors that can be directly
//   handled by the agent without having to stop it.
//
// Each level catches unhandled errors or panics of lower levels:
// - When the agent's main loop fails, it is caught and returned to the upper
//   level to try to send it in a separate safe call, as the agent is no
//   longer considered reliable.
// - If a panic occurs in this overall agent error handling, everything is
//   considered unreliable and therefore aborted.
// - Otherwise, the overall agent initialization and main loop is re-executed
//   with a backoff sleep.
// - If this backoff-retry loop fails, the outer-most safe goroutine captures
//   it and silently return.
func (instance *agentInstanceType) start() {
	instance.startOnce.Do(func() {
		sqsafe.Go(func() error {
			// Level 1
			// Backoff-sleep loop to retry starting the agent
			// To properly work, this level relies on:
			//   - the backoff.
			//   - the logger.
			//   - the correctness of sub-level error handling (ie. they don't panic).
			// Any panics from these would stop the execution of this level.
			backoff := sqtime.NewBackoff(time.Second, time.Hour, 2)
			logger := plog.NewLogger(plog.Info, os.Stderr, nil)
			for {
				err := sqsafe.Call(func() error {
					// Level 2
					// Agent initialization and serve loop.
					// To properly work, this level relies on:
					//   - the user configuration initialization.
					//   - the agent initialization.
					// Any panics from these would stop the execution and would be returned
					// to the outer level.
					cfg, err := config.New(logger)
					if err != nil {
						logger.Error(sqerrors.Wrap(err, "agent disabled"))
						return nil
					}
					agent := New(cfg)
					if agent == nil {
						return nil
					} else {
						instance.set(agent)
					}

					// Level 3 returns unhandled agent errors or panics
					err = sqsafe.Call(agent.Serve)
					if err == nil {
						return nil
					}
					// Error ignored here
					_ = sqsafe.Call(func() error {
						// Send the error with a direct HTTP POST call without using the
						// failed agent, but rather using the standard library's default
						// HTTP client.
						TrySendAppException(logger, cfg, err)
						return nil
					})

					if panicErr, ok := err.(*sqsafe.PanicError); ok {
						// agent.Serve() panic-ed: return the wrapped error in order to retry
						return panicErr.Unwrap()
					}
					return err
				})

				// No error: regular exit case of the agent.
				if err == nil {
					return nil
				}

				if _, ok := err.(*sqsafe.PanicError); ok {
					// Unexpected level 2 panic from its requirements: stop retrying as it
					// is no longer reliable.
					logger.Error(err)
					return err
				}

				// An unhandled error was returned: retry
				logger.Error(errors.Wrap(err, "unexpected agent error"))
				d, max := backoff.Next()
				if max {
					logger.Error(errors.New("agent stopped: maximum agent retries reached"))
					break
				}
				logger.Error(errors.Errorf("retrying to start the agent in %s", d))
				time.Sleep(d)
			}
			return nil
		}, nil)
	})
}

type AgentType struct {
	logger            plog.DebugLevelLogger
	eventMng          *eventManager
	metrics           *metrics.Engine
	staticMetrics     staticMetrics
	ctx               context.Context
	cancel            context.CancelFunc
	isDone            chan struct{}
	config            *config.Config
	appInfo           *app.Info
	client            *backend.Client
	actors            *actor.Store
	rules             *rule.Engine
	piiScrubber       *sqsanitize.Scrubber
	runningAccessLock sync.RWMutex
	running           bool
	performanceBudget time.Duration
	errLoggerChan     chan error
}

type staticMetrics struct {
	sdkUserLoginSuccess,
	sdkUserLoginFailure,
	sdkUserSignup,
	allowedIP,
	allowedPath,
	callCounts *metrics.TimeHistogram
	requestTime, sqreenTime, sqreenOverheadRate *metrics.PerfHistogram
}

// Error channel buffer length.
const errorChanBufferLength = 256

func New(cfg *config.Config) *AgentType {
	errLoggerChan := make(chan error, errorChanBufferLength)
	logger := plog.WithOptionalBackoff(plog.NewLogger(cfg.LogLevel(), os.Stderr, errLoggerChan))

	agentVersion := version.Version()
	logger.Infof("go agent v%s", agentVersion)

	if cfg.Disabled() {
		logger.Infof("agent disabled by the configuration")
		return nil
	}

	metrics := metrics.NewEngine()

	publicKey, err := rule.NewECDSAPublicKey(config.PublicKey)
	if err != nil {
		logger.Error(sqerrors.Wrap(err, "ecdsa public key"))
		return nil
	}
	rulesEngine := rule.NewEngine(logger, nil, metrics, publicKey, perfHistogramUnit, perfHistogramBase, perfHistogramPeriod)

	// Early health checking
	if err := rulesEngine.Health(agentVersion); err != nil {
		message := fmt.Sprintf("agent disabled: %s", err)
		backend.SendAgentMessage(logger, cfg, message)
		logger.Info(message)
		return nil
	}

	if waf.Version() == nil {
		message := "in-app waf disabled: cgo was disabled during the program compilation while required by the in-app waf"
		backend.SendAgentMessage(logger, cfg, message)
		logger.Info("agent: ", message)
	}

	// TODO: remove this SDK metrics period config when the corresponding js rule
	//  is supported
	sdkMetricsPeriod := time.Duration(cfg.SDKMetricsPeriod()) * time.Second
	logger.Debugf("agent: using sdk metrics store time period of %s", sdkMetricsPeriod)

	piiScrubber := sqsanitize.NewScrubber(cfg.StripSensitiveKeyRegexp(), cfg.StripSensitiveValueRegexp(), config.ScrubberRedactedString)

	client, err := backend.NewClient(cfg.BackendHTTPAPIBaseURL(), cfg.BackendHTTPAPIProxy(), logger)
	if err != nil {
		logger.Error(sqerrors.Wrap(err, "agent: could not create the backend client"))
		return nil
	}

	sq, err := metrics.PerfHistogram("sq", perfHistogramUnit, perfHistogramBase, perfHistogramPeriod)
	if err != nil {
		logger.Error(sqerrors.Wrap(err, "`sq` performance histogram constructor error"))
	}
	req, err := metrics.PerfHistogram("req", perfHistogramUnit, perfHistogramBase, perfHistogramPeriod)
	if err != nil {
		logger.Error(sqerrors.Wrap(err, "`req` performance histogram constructor error"))
	}

	const (
		sqreenOverheadRateBase = 1.3
		sqreenOverheadRateUnit = 1.0
	)
	sqOverheadRate, err := metrics.PerfHistogram("pct", sqreenOverheadRateUnit, sqreenOverheadRateBase, perfHistogramPeriod)
	if err != nil {
		logger.Error(sqerrors.Wrap(err, "`pct` performance histogram constructor error"))
	}

	// AgentType graceful stopping using context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentType{
		logger:        logger,
		errLoggerChan: errLoggerChan,
		isDone:        make(chan struct{}),
		metrics:       metrics,
		staticMetrics: staticMetrics{
			sdkUserLoginSuccess: metrics.TimeHistogram("sdk-login-success", sdkMetricsPeriod, 60000),
			sdkUserLoginFailure: metrics.TimeHistogram("sdk-login-fail", sdkMetricsPeriod, 60000),
			sdkUserSignup:       metrics.TimeHistogram("sdk-signup", sdkMetricsPeriod, 60000),
			allowedIP:           metrics.TimeHistogram("whitelisted", sdkMetricsPeriod, 60000),
			allowedPath:         metrics.TimeHistogram("whitelisted_paths", sdkMetricsPeriod, 60000),
			requestTime:         req,
			sqreenTime:          sq,
			sqreenOverheadRate:  sqOverheadRate,
		},
		ctx:         ctx,
		cancel:      cancel,
		config:      cfg,
		appInfo:     app.NewInfo(logger),
		client:      client,
		actors:      actor.NewStore(logger),
		rules:       rulesEngine,
		piiScrubber: piiScrubber,
	}
}

type AgentNotRunningError struct{}

func (AgentNotRunningError) Error() string {
	return "agent not running"
}

func (a *AgentType) sendClosedHTTPProtectionContext(ctx http_protection_types.ClosedProtectionContextFace) {
	if !a.isRunning() {
		a.logger.Debug("agent not running: ignoring the closed http protection context")
		return
	}

	// User events are not part of the request record
	events := ctx.Events()
	for _, event := range events.UserEvents {
		a.addUserEvent(event)
	}

	start := ctx.Start()
	duration := ctx.Duration()
	finish := start.Add(duration)

	// TODO: enforce start as the current time for the performance metrics?
	req := float64(duration.Nanoseconds()) / float64(time.Millisecond)
	if err := a.staticMetrics.requestTime.Add(req); err != nil {
		a.logger.Error(sqerrors.Wrap(err, "could not add the request execution time"))
	}

	sq := float64(ctx.SqreenTime().Nanoseconds()) / float64(time.Millisecond)
	if err := a.staticMetrics.sqreenTime.Add(sq); err != nil {
		a.logger.Error(sqerrors.Wrap(err, "could not add sqreen's execution time"))
	}

	if overheadRate, err := overheadRate(req, sq); err == nil {
		if err := a.staticMetrics.sqreenOverheadRate.Add(overheadRate); err != nil {
			a.logger.Error(sqerrors.Wrap(err, "could not add sqreen overhead rate"))
		}
	}

	event := newClosedHTTPRequestContextEvent(a.RulespackID(), start, finish, ctx.Response(), ctx.Request(), events)
	if !event.shouldSend() {
		return
	}
	a.eventMng.send(event)
}

func overheadRate(req float64, sq float64) (rate float64, err error) {
	if req <= 0 || math.IsNaN(req) || math.IsInf(req, 0) {
		return 0, sqerrors.Errorf("unexpected req value `%v`", req)
	}
	if sq <= 0 || math.IsNaN(sq) || math.IsInf(sq, 0) {
		return 0, sqerrors.Errorf("unexpected sq value `%v`", sq)
	}
	if req < sq {
		return 0, sqerrors.Errorf("unexpected req `%v` and sq `%v` values: req should be greater or equal to sq", req, sq)
	}
	if req == sq {
		return 100, nil
	}

	userTime := req - sq
	if userTime == 0 {
		return 100, nil
	}
	return 100 * sq / userTime, nil
}

type withNotificationError struct {
	error
}

func (e *withNotificationError) Unwrap() error { return e.error }

func (a *AgentType) Serve() error {
	defer func() {
		// Signal we are done
		close(a.isDone)
		a.logger.Info("agent stopped")
	}()

	token := a.config.BackendHTTPAPIToken()
	appName := a.config.AppName()
	appLoginRes, err := appLogin(a.ctx, a.logger, a.client, token, appName, a.appInfo, a.config.DisableSignalBackend())
	if err != nil {
		if xerrors.Is(err, context.Canceled) {
			a.logger.Debug(err)
			return nil
		}
		if xerrors.As(err, &LoginError{}) {
			a.logger.Info(err)
			return nil
		}
		return err
	}

	// Load the rulepack side car
	a.rules.SetRules(appLoginRes.PackID, appLoginRes.Rules)
	// Load the actionpack side car
	if err := a.actors.SetActions(appLoginRes.Actions); err != nil {
		a.logger.Error(sqerrors.Wrap(err, "could not load the list of actions taken from the login response"))
	}

	// Create the command manager to process backend commands
	commandMng := NewCommandManager(a, a.logger)
	// Process commands that may have been received at login.
	commandResults := commandMng.Do(appLoginRes.Commands)

	heartbeat := time.Duration(appLoginRes.Features.HeartbeatDelay) * time.Second
	if heartbeat == 0 {
		heartbeat = config.BackendHTTPAPIDefaultHeartbeatDelay
	}

	batchSize := int(appLoginRes.Features.BatchSize)
	if batchSize == 0 {
		batchSize = config.EventBatchMaxEventsPerHeartbeat
	}
	maxStaleness := time.Duration(appLoginRes.Features.MaxStaleness) * time.Second
	if maxStaleness == 0 {
		maxStaleness = config.EventBatchMaxStaleness
	}

	// start the event manager's loop
	queueLength := appLoginRes.Features.EventQueueLength
	if queueLength == 0 {
		queueLength = config.EventQueueDefaultLength
	}
	a.eventMng = newEventManager(a, queueLength, uint32(runtime.NumCPU()), batchSize, maxStaleness)
	a.eventMng.Start()

	a.setRunning(true)
	defer a.setRunning(false)

	a.logger.Debugf("agent: heartbeat ticker set to %s", heartbeat)
	ticker := time.Tick(heartbeat)
	a.logger.Info("agent: up and running")

	// start the agent main loop
	for {
		select {
		case <-ticker:
			a.logger.Debug("heartbeat")

			appBeatReq := api.AppBeatRequest{
				Metrics:        newMetricsAPIAdapter(a.logger, a.metrics.ReadyMetrics()),
				CommandResults: commandResults,
			}

			appBeatRes, err := a.client.AppBeat(a.ctx, &appBeatReq)
			if err != nil {
				a.logger.Error(sqerrors.Wrap(err, "heartbeat failed"))
				continue
			}

			// Perform commands that may be requested.
			commandResults = commandMng.Do(appBeatRes.Commands)

		case <-a.ctx.Done():
			// The context was canceled because of a interrupt signal, logout and
			// return.
			err := a.client.AppLogout()
			if err != nil {
				a.logger.Debug("logout failed: ", err)
				return nil
			}
			a.logger.Debug("successfully logged out")
			return nil

		case err := <-a.eventMng.errChan:
			if err == nil {
				continue
			}
			// Unexpected error from the event manager's loop as it should stop
			// when the agent stops.
			return err

		case err := <-a.errLoggerChan:
			// Logged errors.
			if xerrors.As(err, &withNotificationError{}) {
				t, ok := sqerrors.Timestamp(err)
				if !ok {
					t = time.Now()
				}
				_ = a.client.SendAgentMessage(a.ctx, t, err.Error(), nil)
			}
			a.addExceptionEvent(NewExceptionEvent(err, a.RulespackID()))
		}
	}
}

func (a *AgentType) EnableInstrumentation() (string, error) {
	var id string
	if a.rules.Count() == 0 {
		var err error
		id, err = a.ReloadRules()
		if err != nil {
			return "", err
		}
	} else {
		id = a.RulespackID()
	}
	a.rules.Enable()
	a.setRunning(true)
	a.logger.Debug("agent: enabled")
	return id, nil
}

// DisableInstrumentation disables the agent instrumentation, which includes for
// now the SDK.
func (a *AgentType) DisableInstrumentation() error {
	a.setRunning(false)
	a.rules.Disable()
	err := a.actors.SetActions(nil)
	a.logger.Debug("agent: disabled")
	return err
}

func (a *AgentType) ReloadActions() error {
	actions, err := a.client.ActionsPack()
	if err != nil {
		a.logger.Error(err)
		return err
	}
	return a.actors.SetActions(actions.Actions)
}

func (a *AgentType) SendAppBundle() error {
	deps, sig, _ := a.appInfo.Dependencies()
	bundleDeps := make([]api.AppDependency, len(deps))
	for i, dep := range deps {
		bundleDeps[i] = api.AppDependency{
			Name:    dep.Path,
			Version: dep.Version,
		}
	}
	bundle := api.AppBundle{
		Signature:    sig,
		Dependencies: bundleDeps,
	}

	return a.client.SendAppBundle(&bundle)
}

func (a *AgentType) SetCIDRIPPasslist(cidrs []string) error {
	return a.actors.SetCIDRIPPasslist(cidrs)
}

func (a *AgentType) SetPathPasslist(paths []string) error {
	a.actors.SetPathPasslist(paths)
	return nil
}

func (a *AgentType) ReloadRules() (string, error) {
	rulespack, err := a.client.RulesPack()
	if err != nil {
		a.logger.Error(err)
		return "", err
	}

	// Insert local rules if any
	if localRulesJSON := a.config.LocalRulesFile(); localRulesJSON != "" {
		buf, err := ioutil.ReadFile(localRulesJSON)
		if err == nil {
			var localRules []api.Rule
			err = json.Unmarshal(buf, &localRules)
			if err == nil {
				rulespack.Rules = append(rulespack.Rules, localRules...)
			}
		}
		if err != nil {
			a.logger.Error(sqerrors.Wrap(err, "config: could not read the local rules file"))
		}
	}

	a.rules.SetRules(rulespack.PackID, rulespack.Rules)
	return rulespack.PackID, nil
}

func (a *AgentType) SetPerformanceBudget(budget float64) error {
	a.performanceBudget = time.Duration(budget * float64(time.Millisecond))
	return nil
}

func (a *AgentType) gracefulStop() {
	a.cancel()
	<-a.isDone
}

type eventManager struct {
	agent          *AgentType
	maxBatchLength int
	eventsChan     chan Event
	maxStaleness   time.Duration
	stats          *metrics.TimeHistogram
	maxGoroutines  uint32
	nbGoroutines   uint32
	errChan        chan error
}

func newEventManager(agent *AgentType, queueLength uint, maxGoroutines uint32, maxBatchLength int, maxStaleness time.Duration) *eventManager {
	stats := agent.metrics.TimeHistogram("event_management", time.Minute, 10)
	return &eventManager{
		agent:          agent,
		eventsChan:     make(chan Event, queueLength),
		maxBatchLength: maxBatchLength,
		maxStaleness:   maxStaleness,
		stats:          stats,
		maxGoroutines:  maxGoroutines,
		errChan:        make(chan error, maxGoroutines),
	}
}

func (m *eventManager) send(e Event) {
	select {
	case m.eventsChan <- e:
		m.stats.Add("queue_ingress", 1)
	default:
		// The channel buffer is full - drop this event
		m.scaleUp()
		m.stats.Add("queue_dropped", 1)
	}
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}

func (m *eventManager) ErrChan() <-chan error {
	return m.errChan
}

func (m *eventManager) Start() {
	atomic.StoreUint32(&m.nbGoroutines, 1)
	m.start()
}

func (m *eventManager) start() {
	sqsafe.Go(func() error {
		m.loop()
		return nil
	}, m.errChan)
}

func (m *eventManager) scaleUp() {
	if n := atomic.LoadUint32(&m.nbGoroutines); n == 0 || n == m.maxGoroutines {
		return
	}
	atomic.AddUint32(&m.nbGoroutines, 1)
	m.start()
	m.stats.Add("scale", 1)
}

func (m *eventManager) loop() {
	var (
		// We can't create a stopped timer so we initialize it with a large value
		// of 24 hours and stop it immediately. Calls to Reset() will correctly
		// set the configured timer value.
		stalenessTimer = time.NewTimer(24 * time.Hour)
		stalenessChan  <-chan time.Time
	)
	stopTimer(stalenessTimer)
	defer stopTimer(stalenessTimer)

	ctx := m.agent.ctx
	client := m.agent.client
	batch := make([]Event, 0, m.maxBatchLength)
	req := &api.BatchRequest{
		Batch: make([]api.BatchRequest_Event, 0, cap(batch)),
	}
	for {
		select {
		case <-ctx.Done():
			return

		case <-stalenessChan:
			m.agent.logger.Debug("event batch data staleness reached")
			m.sendBatch(ctx, client, batch, req)
			batch = batch[0:0]
			stalenessChan = nil

		case event := <-m.eventsChan:
			batch = append(batch, event)
			m.stats.Add("queue_egress", 1)
			m.agent.logger.Debugf("event `%T` added to the event batch", event)

			batchLen := len(batch)
			switch {
			case batchLen == 1:
				stalenessTimer.Reset(m.maxStaleness)
				stalenessChan = stalenessTimer.C
				m.agent.logger.Debug("batching events for ", m.maxStaleness)

			case batchLen >= m.maxBatchLength:
				// No more room in the batch
				m.agent.logger.Debugf("sending the batch of %d events", batchLen)
				m.sendBatch(ctx, client, batch, req)
				batch = batch[0:0]
				stalenessChan = nil
				stopTimer(stalenessTimer)
			}
		}
	}
}

func (m *eventManager) sendBatch(ctx context.Context, client *backend.Client, batch []Event, req *api.BatchRequest) {
	defer func() {
		req.Batch = req.Batch[0:0]
	}()
	for _, e := range batch {
		var event api.BatchRequest_EventFace
		switch actual := e.(type) {
		case *closedHTTPRequestContextEvent:
			cfg := m.agent.config
			adapter := newProtectedHTTPRequestEventAPIAdapter(actual, cfg.StripHTTPReferer(), cfg.HTTPClientIPHeader())
			event = api.RequestRecordEvent{api.NewRequestRecordFromFace(adapter)}
		case *ExceptionEvent:
			event = api.NewExceptionEventFromFace(actual)
		}

		// Scrub the value, along with the set of scrubbed string values.
		if _, err := m.agent.piiScrubber.Scrub(event, nil); err != nil {
			// Only log this unexpected error and keep the event that may have been
			// partially scrubbed.
			m.agent.logger.Error(errors.Wrap(err, "could not scrub the event"))
		}
		req.Batch = append(req.Batch, *api.NewBatchRequest_EventFromFace(event))
	}

	// Send the batch.
	if err := client.Batch(ctx, req); err != nil {
		m.stats.Add("backend_dropped", uint64(len(req.Batch)))
	} else {
		m.stats.Add("backend_egress", uint64(len(req.Batch)))
	}
}

func (a *AgentType) setRunning(r bool) {
	a.runningAccessLock.Lock()
	defer a.runningAccessLock.Unlock()
	a.running = r
}

func (a *AgentType) isRunning() bool {
	a.runningAccessLock.RLock()
	defer a.runningAccessLock.RUnlock()
	return a.running
}

func (a *AgentType) addExceptionEvent(e *ExceptionEvent) {
	if !a.isRunning() {
		return
	}
	a.eventMng.send(e)
}

func (a *AgentType) RulespackID() string {
	return a.rules.PackID()
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/actor"
	"github.com/sqreen/go-agent/agent/internal/app"
	"github.com/sqreen/go-agent/agent/internal/backend"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqsafe"
	"github.com/sqreen/go-agent/agent/sqlib/sqtime"
	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
	"golang.org/x/xerrors"
)

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
//   it an silently return.
func Start() {
	_ = sqsafe.Go(func() error {
		// Level 1
		// Backoff-sleep loop to retry starting the agent
		// To properly work, this level relies on:
		//   - the backoff.
		//   - the logger.
		//   - the correctness of sub-level error handling (ie. they don't panic).
		// Any panics from these would stop the execution of this level.
		backoff := sqtime.NewBackoff(time.Second, time.Hour, 2)
		logger := plog.NewLogger(plog.Debug, os.Stderr, 0)
		for {
			err := sqsafe.Call(func() error {
				// Level 2
				// Agent initialization and serve loop.
				// To properly work, this level relies on:
				//   - the user configuration initialization.
				//   - the agent initialization.
				// Any panics from these would stop the execution and would be returned
				// to the outer level.
				cfg := config.New(logger)
				agent := New(cfg)
				if agent == nil {
					return nil
				}
				// Level 3 returns unhandled agent errors or panics
				err := sqsafe.Call(agent.Serve)
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
	})
}

type Agent struct {
	logger     *plog.Logger
	eventMng   *eventManager
	metricsMng *metricsManager
	ctx        context.Context
	cancel     context.CancelFunc
	isDone     chan struct{}
	config     *config.Config
	appInfo    *app.Info
	client     *backend.Client
	actors     *actor.Store
	rules      *rule.Engine
}

// Error channel buffer length.
const errorChanBufferLength = 256

func New(cfg *config.Config) *Agent {
	logger := plog.NewLogger(plog.ParseLogLevel(cfg.LogLevel()), os.Stderr, errorChanBufferLength)

	if cfg.Disable() {
		logger.Info("agent disabled by configuration")
		return nil
	}

	// Agent graceful stopping using context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		logger:     logger,
		isDone:     make(chan struct{}),
		metricsMng: newMetricsManager(ctx, logger),
		ctx:        ctx,
		cancel:     cancel,
		config:     cfg,
		appInfo:    app.NewInfo(logger),
		client:     backend.NewClient(cfg.BackendHTTPAPIBaseURL(), cfg, logger),
		actors:     actor.NewStore(logger),
		rules:      rule.NewEngine(logger),
	}
}

func (a *Agent) NewRequestRecord(req *http.Request) types.RequestRecord {
	clientIP := getClientIP(req, a.config)
	whitelisted, matched, err := a.actors.IsIPWhitelisted(clientIP)
	if err != nil {
		a.logger.Error(err)
		whitelisted = false
	}
	if whitelisted {
		a.addWhitelistEvent(matched)
		return &WhitelistedHTTPRequestRecord{
		}
	}
	return &HTTPRequestRecord{
		request:  req,
		agent:    a,
		clientIP: clientIP,
	}
}

func (a *Agent) Serve() error {
	defer func() {
		// Signal we are done
		close(a.isDone)
		a.logger.Info("agent stopped")
	}()

	token := a.config.BackendHTTPAPIToken()
	appName := a.config.AppName()
	appLoginRes, err := appLogin(a.ctx, a.logger, a.client, token, appName, a.appInfo)
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

	// Create the command manager to process backend commands
	commandMng := NewCommandManager(a, a.logger)
	// Process commands that may have been received at login.
	commandResults := commandMng.Do(appLoginRes.Commands)

	heartbeat := time.Duration(appLoginRes.Features.HeartbeatDelay) * time.Second
	if heartbeat == 0 {
		heartbeat = config.BackendHTTPAPIDefaultHeartbeatDelay
	}

	a.logger.Info("up and running - heartbeat set to ", heartbeat)
	ticker := time.Tick(heartbeat)

	batchSize := int(appLoginRes.Features.BatchSize)
	if batchSize == 0 {
		batchSize = config.MaxEventsPerHeatbeat
	}
	maxStaleness := time.Duration(appLoginRes.Features.MaxStaleness) * time.Second
	if maxStaleness == 0 {
		maxStaleness = config.EventBatchMaxStaleness
	}

	// Start the event manager's loop
	a.eventMng = newEventManager(a, batchSize, maxStaleness)
	eventMngErrChan := sqsafe.Go(func() error {
		a.eventMng.Loop(a.ctx, a.client)
		return nil
	})

	// Start the heartbeat's loop
	for {
		select {
		case <-ticker:
			a.logger.Debug("heartbeat")

			metrics := a.metricsMng.getObservations()
			appBeatReq := api.AppBeatRequest{
				Metrics:        metrics,
				CommandResults: commandResults,
			}

			appBeatRes, err := a.client.AppBeat(&appBeatReq)
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

		case err := <-eventMngErrChan:
			// Unexpected error from the event manager's loop as it should stop
			// when the agent stops.
			return err

		case err := <-a.logger.ErrChan():
			// Unhandled errors that were logged.
			a.AddExceptionEvent(NewExceptionEvent(err, a.RulespackID()))
		}
	}
}

func (a *Agent) InstrumentationEnable() error {
	if err := a.RulesReload(); err != nil {
		return err
	}
	a.rules.Enable()
	sdk.SetAgent(a)
	a.logger.Info("instrumentation enabled")
	return nil
}

// InstrumentationDisable disables the agent instrumentation, which includes for
// now the SDK.
func (a *Agent) InstrumentationDisable() error {
	sdk.SetAgent(nil)
	a.rules.Disable()
	err := a.actors.SetActions(nil)
	a.logger.Info("instrumentation disabled")
	return err
}

func (a *Agent) ActionsReload() error {
	actions, err := a.client.ActionsPack()
	if err != nil {
		a.logger.Error(err)
		return err
	}

	return a.actors.SetActions(actions.Actions)
}

func (a *Agent) SetCIDRWhitelist(cidrs []string) error {
	return a.actors.SetCIDRWhitelist(cidrs)
}

func (a *Agent) RulesReload() error {
	rulespack, err := a.client.RulesPack()
	if err != nil {
		a.logger.Error(err)
		return err
	}

	// Insert local rules if any
	localRulesJSON := a.config.LocalRulesFile()
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

	a.rules.SetRules(rulespack.PackID, rulespack.Rules)
	return nil
}

func (a *Agent) GracefulStop() {
	if a.config.Disable() {
		return
	}
	a.cancel()
	<-a.isDone
}

type eventManager struct {
	agent        *Agent
	count        int
	eventsChan   chan Event
	maxStaleness time.Duration
}

func newEventManager(agent *Agent, count int, maxStaleness time.Duration) *eventManager {
	return &eventManager{
		agent:        agent,
		eventsChan:   make(chan Event, count*100),
		count:        count,
		maxStaleness: maxStaleness,
	}
}

func (m *eventManager) add(r Event) {
	select {
	case m.eventsChan <- r:
		return
	default:
		// The channel buffer is full - drop this new event
		m.agent.logger.Debug("event channel is full, dropping the event")
	}
}

func stopTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}

func (m *eventManager) Loop(ctx context.Context, client *backend.Client) {
	var (
		stalenessTimer = time.NewTimer(m.maxStaleness)
		stalenessChan  <-chan time.Time
	)
	defer stopTimer(stalenessTimer)
	stopTimer(stalenessTimer)

	batch := make([]Event, 0, m.count)
	for {
		select {
		case <-ctx.Done():
			return

		case <-stalenessChan:
			m.agent.logger.Debug("event batch data staleness reached")
			m.send(client, batch)
			batch = batch[0:0]
			stalenessChan = nil

		case event := <-m.eventsChan:
			batch = append(batch, event)
			m.agent.logger.Debugf("new event `%T` added to the event batch", event)

			batchLen := len(batch)
			switch {
			case batchLen == 1:
				stalenessTimer.Reset(m.maxStaleness)
				stalenessChan = stalenessTimer.C
				m.agent.logger.Debug("batching events for ", m.maxStaleness)

			case batchLen >= m.count:
				// No more room in the batch
				m.agent.logger.Debugf("sending the batch of %d events", batchLen)
				m.send(client, batch)
				batch = batch[0:0]
				stalenessChan = nil
				stopTimer(stalenessTimer)
			}
		}
	}
}

func (m *eventManager) send(client *backend.Client, batch []Event) {
	req := api.BatchRequest{
		Batch: make([]api.BatchRequest_Event, 0, len(batch)),
	}

	for _, e := range batch {
		var event api.BatchRequest_EventFace
		switch actual := e.(type) {
		case *HTTPRequestRecordEvent:
			event = (*api.RequestRecordEvent)(api.NewRequestRecordFromFace(actual))
		case *ExceptionEvent:
			event = api.NewExceptionEventFromFace(actual)
		}
		req.Batch = append(req.Batch, *api.NewBatchRequest_EventFromFace(event))
	}

	// Send the batch.
	if err := client.Batch(&req); err != nil {
		// Log the error and drop the batch
		m.agent.logger.Error(errors.Wrap(err, "could not send the event batch"))
	}
}

func (a *Agent) AddHTTPRequestRecordEvent(e *HTTPRequestRecordEvent) {
	if a.config.Disable() || a.eventMng == nil {
		// Disabled or not yet initialized agent
		return
	}
	a.eventMng.add(e)
}

func (a *Agent) AddExceptionEvent(e *ExceptionEvent) {
	if a.config.Disable() || a.eventMng == nil {
		// Disabled or not yet initialized agent
		return
	}
	a.eventMng.add(e)
}

func (a *Agent) RulespackID() string {
	return a.rules.PackID()
}

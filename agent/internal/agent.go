package internal

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/actor"
	"github.com/sqreen/go-agent/agent/internal/app"
	"github.com/sqreen/go-agent/agent/internal/backend"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
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
					// TODO: try to send the error with a direct bare HTTP POST call not
					//  using the agent but relying on the configuration in order to get
					//  the token.
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
}

func New(cfg *config.Config) *Agent {
	if cfg.Disable() {
		return nil
	}

	logger := plog.NewLogger(plog.ParseLogLevel(cfg.LogLevel()), os.Stderr, 0)

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
	}
}

func (a *Agent) NewRequestRecord(req *http.Request) types.RequestRecord {
	return &HTTPRequestRecord{
		request: req,
		agent:   a,
	}
}

func (a *Agent) Serve() error {
	defer func() {
		// Signal we are done
		close(a.isDone)
		a.logger.Info("agent successfully stopped")
	}()

	token := a.config.BackendHTTPAPIToken()
	appName := a.config.AppName()
	appLoginRes, err := appLogin(a.ctx, a.logger, a.client, token, appName, a.appInfo)
	if err != nil {
		if xerrors.Is(err, context.Canceled) {
			return nil
		}
		// TODO: add the error into the event batch
		return err
	}

	// Create the command manager to process backend commands
	commandMng := NewCommandManager(a, a.logger)
	// Process commands that may have been received on login.
	commandResults := commandMng.Do(appLoginRes.Commands)

	heartbeat := time.Duration(appLoginRes.Features.HeartbeatDelay) * time.Second
	if heartbeat == 0 {
		heartbeat = config.BackendHTTPAPIDefaultHeartbeatDelay
	}

	a.logger.Info("up and running - heartbeat set to ", heartbeat)
	ticker := time.Tick(heartbeat)

	rulespackId := appLoginRes.PackId
	batchSize := int(appLoginRes.Features.BatchSize)
	if batchSize == 0 {
		batchSize = config.MaxEventsPerHeatbeat
	}
	maxStaleness := time.Duration(appLoginRes.Features.MaxStaleness) * time.Second
	if maxStaleness == 0 {
		maxStaleness = config.EventBatchMaxStaleness
	}

	// Start the event manager's loop
	a.eventMng = newEventManager(a, rulespackId, batchSize, maxStaleness)
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
				a.logger.Debug("heartbeat failed: ", err)
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
			// TODO: add the error to the error batch
			return err
		}
	}
}

func (a *Agent) InstrumentationEnable() error {
	sdk.SetAgent(a)
	a.logger.Info("instrumentation enabled")
	return nil
}

// InstrumentationDisable disables the agent instrumentation, which includes for
// now the SDK.
func (a *Agent) InstrumentationDisable() error {
	sdk.SetAgent(nil)
	a.logger.Info("instrumentation disabled")
	err := a.actors.SetActions(nil)
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

func (a *Agent) GracefulStop() {
	if a.config.Disable() {
		return
	}
	a.cancel()
	<-a.isDone
}

type eventManager struct {
	agent        *Agent
	req          api.BatchRequest
	rulespackID  string
	count        int
	eventsChan   chan (*httpRequestRecord)
	reqLock      sync.Mutex
	maxStaleness time.Duration
}

func newEventManager(agent *Agent, rulespackID string, count int, maxStaleness time.Duration) *eventManager {
	return &eventManager{
		agent:        agent,
		eventsChan:   make(chan (*httpRequestRecord), count),
		count:        count,
		rulespackID:  rulespackID,
		maxStaleness: maxStaleness,
	}
}

func (m *eventManager) add(r *httpRequestRecord) {
	select {
	case m.eventsChan <- r:
		return
	default:
		// The channel buffer is full - drop this new event
		m.agent.logger.Debug("event channel is full, dropping the event")
	}
}

func (m *eventManager) Loop(ctx context.Context, client *backend.Client) {
	var isFull, isSent chan struct{}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-m.eventsChan:
			m.agent.logger.Debug("new event")
			event.SetRulespackId(m.rulespackID)
			m.reqLock.Lock()
			m.req.Batch = append(m.req.Batch, api.BatchRequest_Event{
				EventType: "request_record",
				Event:     api.Struct{api.NewRequestRecordFromFace(event)},
			})
			batchLen := len(m.req.Batch)
			m.reqLock.Unlock()
			if batchLen == 1 {
				// First event of this batch.
				// Given the amount of external events, it is easier to create a
				// goroutine to select{} one of them.
				m.agent.logger.Debug("batching event data for ", m.maxStaleness)
				isFull = make(chan struct{})
				isSent = make(chan struct{}, 1)
				// FIXME: get rid of this goroutine
				_ = sqsafe.Go(func() error {
					select {
					case <-ctx.Done():
					case <-time.After(m.maxStaleness):
						m.agent.logger.Debug("event batch data staleness reached")
					case <-isFull:
						m.agent.logger.Debug("event batch is full")
					}
					m.send(client)
					m.agent.logger.Debug("event batch sent")
					close(isSent)
					return nil
				})
			} else if batchLen >= m.count {
				// No more room in the batch
				close(isFull)
				<-isSent
				// Block until it is sent. There are many reasons to this, but it is
				// mainly to avoid running this loop and the sender goroutines
				// concurrently. For example, it allows to make sure the previous
				// len(m.req.Batch) == 1 to be observable.
			}
		}
	}
}

func (m *eventManager) send(client *backend.Client) {
	m.reqLock.Lock()
	defer m.reqLock.Unlock()
	// Send the batch.
	if err := client.Batch(&m.req); err != nil {
		// Log the error and drop the batch
		m.agent.logger.Error(errors.Wrap(err, "could not send an event batch"))
	}
	m.req.Batch = m.req.Batch[0:0]
}

func (a *Agent) addRecord(r *httpRequestRecord) {
	if a.config.Disable() || a.eventMng == nil {
		// Disabled or not yet initialized agent
		return
	}
	a.eventMng.add(r)
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sqreen/go-agent/agent/internal/actor"
	"github.com/sqreen/go-agent/agent/internal/app"
	"github.com/sqreen/go-agent/agent/internal/backend"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
)

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

func New() *Agent {
	logger := plog.NewLogger("agent", nil)
	cfg := config.New(logger)
	plog.SetLevelFromString(cfg.LogLevel())
	plog.SetOutput(os.Stderr)

	if cfg.Disable() {
		return nil
	}

	// Agent graceful stopping using context cancelation.
	ctx, cancel := context.WithCancel(context.Background())

	client, err := backend.NewClient(cfg.BackendHTTPAPIBaseURL(), cfg, logger)
	if err != nil {
		logger.Error(err)
		return nil
	}

	return &Agent{
		logger:     logger,
		isDone:     make(chan struct{}),
		metricsMng: newMetricsManager(ctx, logger),
		ctx:        ctx,
		cancel:     cancel,
		config:     cfg,
		appInfo:    app.NewInfo(logger),
		client:     client,
		actors:     actor.NewStore(logger),
	}
}

func (a *Agent) Start() {
	go a.start()
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

func (a *Agent) start() {
	defer func() {
		err := recover()
		if err != nil {
			a.logger.Error("agent stopped: ", err)
		} else {
			a.logger.Info("agent successfully stopped")
		}
		// Signal we are done
		close(a.isDone)
	}()

	token := a.config.BackendHTTPAPIToken()
	appName := a.config.AppName()
	appLoginRes, err := appLogin(a.ctx, a.logger, a.client, token, appName, a.appInfo)
	if checkErr(err, a.logger) {
		return
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
	go a.eventMng.Loop(a.ctx, a.client)

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
				a.logger.Error("heartbeat failed: ", err)
				continue
			}

			// Perform commands that may be requested.
			commandResults = commandMng.Do(appBeatRes.Commands)

		case <-a.ctx.Done():
			// The context was canceled because of a interrupt signal, logout and
			// return.
			err := a.client.AppLogout()
			if err != nil {
				a.logger.Error("logout failed: ", err)
				return
			}
			a.logger.Debug("successfully logged out")
			return
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

func (a *Agent) SetCIDRWhitelist(cidrs []string) error {
	return a.actors.SetCIDRWhitelist(cidrs)
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
				go func() {
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
				}()
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
		m.agent.logger.Error("could not send an event batch: ", err)
		// drop it
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

// Helper function returning true when having to exit the agent and panic-ing
// when the error is something else than context cancelation.
func checkErr(err error, logger *plog.Logger) (exit bool) {
	if err != nil {
		if err != context.Canceled {
			logger.Panic(err)
		}
		return true
	}
	return false
}

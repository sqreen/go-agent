package agent

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/sqreen/go-agent/agent/backend"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/agent/config"
	"github.com/sqreen/go-agent/agent/plog"
)

func init() {
	start()
}

func start() {
	if config.Disable() {
		return
	}
	go agent()
}

var (
	logger   = plog.NewLogger("sqreen/agent")
	eventMng *eventManager
	cancel   context.CancelFunc
	isDone   chan struct{}
)

func agent() {
	defer func() {
		err := recover()
		if err != nil {
			logger.Error("agent stopped: ", err)
			return
		}
		logger.Info("agent successfully stopped")
	}()

	plog.SetLevelFromString(config.LogLevel())
	plog.SetOutput(os.Stderr)

	// Agent stopping using context cancelation externally called through the SDK.
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	isDone = make(chan struct{})

	client, err := backend.NewClient(config.BackendHTTPAPIBaseURL())
	if checkErr(err) {
		return
	}

	appLoginRes, err := appLogin(ctx, client)
	if checkErr(err) {
		return
	}

	heartbeat := time.Duration(appLoginRes.Features.HeartbeatDelay) * time.Second
	if heartbeat == 0 {
		heartbeat = config.BackendHTTPAPIDefaultHeartbeatDelay
	}

	logger.Info("up and running - heartbeat set to ", heartbeat)
	ticker := time.Tick(heartbeat)
	sessionID := appLoginRes.SessionId
	rulespackId := appLoginRes.PackId
	batchSize := int(appLoginRes.Features.BatchSize)
	if batchSize == 0 {
		batchSize = config.MaxEventsPerHeatbeat
	}
	maxStaleness := time.Duration(appLoginRes.Features.MaxStaleness) * time.Second
	if maxStaleness == 0 {
		maxStaleness = config.EventBatchMaxStaleness
	}

	eventMng = newEventManager(rulespackId, batchSize, maxStaleness)
	// Start the event manager's loop
	go eventMng.Loop(ctx, client, sessionID)

	// Start the heartbeat's loop
	for {
		select {
		case <-ticker:
			logger.Debug("heartbeat")
			var appBeatReq api.AppBeatRequest
			_, err := client.AppBeat(&appBeatReq, sessionID)
			if err != nil {
				logger.Error("heartbeat failed: ", err)
				continue
			}

		case <-ctx.Done():
			// The context was canceled because of a interrupt signal, logout and
			// return.
			err := client.AppLogout(sessionID)
			if err != nil {
				logger.Error("logout failed: ", err)
				return
			}
			logger.Debug("successfully logged out")
			// Signal we are done
			close(isDone)
			return
		}
	}
}

func GracefulStop() {
	if config.Disable() {
		return
	}
	cancel()
	<-isDone
}

type eventManager struct {
	req          api.BatchRequest
	rulespackID  string
	count        int
	eventsChan   chan (*httpRequestRecord)
	reqLock      sync.Mutex
	maxStaleness time.Duration
}

func newEventManager(rulespackID string, count int, maxStaleness time.Duration) *eventManager {
	return &eventManager{
		eventsChan:   make(chan (*httpRequestRecord), count),
		count:        count,
		rulespackID:  rulespackID,
		maxStaleness: maxStaleness,
	}
}

func (m *eventManager) addEvent(r *httpRequestRecord) {
	select {
	case m.eventsChan <- r:
		return
	default:
		// The channel buffer is full - drop this new event
		logger.Debug("event channel is full, dropping the event")
	}
}

func (m *eventManager) Loop(ctx context.Context, client *backend.Client, sessionID string) {
	var isFull, isSent chan struct{}
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-m.eventsChan:
			logger.Debug("new event")
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
				logger.Debug("batching event data for ", m.maxStaleness)
				isFull = make(chan struct{})
				isSent = make(chan struct{}, 1)
				go func() {
					select {
					case <-ctx.Done():
						return
					case <-time.After(m.maxStaleness):
						logger.Debug("event batch data staleness reached")
					case <-isFull:
						logger.Debug("event batch is full")
					}
					m.send(client, sessionID)
					logger.Debug("event batch sent")
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

func (m *eventManager) send(client *backend.Client, sessionID string) {
	m.reqLock.Lock()
	defer m.reqLock.Unlock()
	// Send the batch.
	if err := client.Batch(&m.req, sessionID); err != nil {
		logger.Error("could not send an event batch: ", err)
		// drop it
	}
	m.req.Batch = m.req.Batch[0:0]
}

func addEvent(r *httpRequestRecord) {
	if eventMng == nil {
		// Disabled or not yet initialized agent
		return
	}
	eventMng.addEvent(r)
}

// Helper function returning true when having to exit the agent and panic-ing
// when the error is something else than context cancelation.
func checkErr(err error) (exit bool) {
	if err != nil {
		if err != context.Canceled {
			logger.Panic(err)
		}
		return true
	}
	return false
}

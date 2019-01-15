package agent

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/sqreen/go-agent/agent/backend"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/agent/config"
	"github.com/sqreen/go-agent/agent/plog"
)

const (
	agentVersion = "0.1.0"
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
	logger     = plog.NewLogger("sqreen/agent")
	eventsChan = make(chan (*httpRequestRecord), config.MaxEventsPerHeatbeat)
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

	// Cleanly stop the agent execution when receiving an interrupt signal
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer signal.Stop(c)
		<-c
		logger.Debug("interrupt signal received: canceling the agent")
		cancel()
		runtime.Gosched()
	}()

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
	batch := (*batch)(&api.BatchRequest{})
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

			nbEvents := batch.processEvents(rulespackId, batchSize)
			if nbEvents > 0 {
				err = client.Batch((*api.BatchRequest)(batch), sessionID)
				if err == nil {
					logger.Debugf("sent %d events", nbEvents)
					batch.Reset()
				} else {
					logger.Error("could not send the event batch: ", err)
				}
			}

		case <-ctx.Done():
			// The context was canceled because of a interrupt signal, logout and
			// return.
			err := client.AppLogout(sessionID)
			if err != nil {
				logger.Error("logout failed: ", err)
				return
			}
			logger.Debug("successfully logged out\n")
			return
		}
	}
}

type batch api.BatchRequest

func (b *batch) processEvents(rulespackId string, count int) int {
	n := 0
	for i := 0; i < count; i++ {
		select {
		case event := <-eventsChan:
			event.SetRulespackId(rulespackId)
			b.Batch = append(b.Batch, api.BatchRequest_Event{
				EventType: "request_record",
				Event:     api.Struct{api.NewRequestRecordFromFace(event)},
			})
			n++
		default:
			break
		}
	}
	return n
}

func (b *batch) Reset() {
	b.Batch = b.Batch[0:0]
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

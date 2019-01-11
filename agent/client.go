package agent

import (
	"context"
	"time"

	"github.com/sqreen/go-agent/agent/app"
	"github.com/sqreen/go-agent/agent/backend"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/agent/config"
)

var (
	backoffRate     = 2.0
	backoffDuration = time.Millisecond
)

// Login to the backend. When the API request fails, retry for ever and after
// sleeping some time.
func appLogin(ctx context.Context, client *backend.Client) (*api.AppLoginResponse, error) {
	procInfo := app.GetProcessInfo()

	appLoginReq := api.AppLoginRequest{
		VariousInfos:    *api.NewAppLoginRequest_VariousInfosFromFace(procInfo),
		BundleSignature: "",
		AgentType:       "golang",
		AgentVersion:    agentVersion,
		OsType:          app.GoBuildTarget(),
		Hostname:        app.Hostname(),
		RuntimeVersion:  app.GoVersion(),
	}

	var (
		appLoginRes *api.AppLoginResponse
		err         error
	)

	var backoff backoff
	for appLoginRes == nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			appLoginRes, err = client.AppLogin(&appLoginReq, config.BackendHTTPAPIToken())
			if err != nil {
				logger.Error(err)
				appLoginRes = nil
				backoff.sleep()
			}
		}
	}

	return appLoginRes, nil
}

type backoff struct {
	duration time.Duration
	fails    uint64
}

func (b *backoff) next() {
	b.fails++
	if b.duration == 0 {
		b.duration = config.BackendHTTPAPIBackoffMinDuration
	} else if b.duration > config.BackendHTTPAPIBackoffMaxDuration {
		b.duration = config.BackendHTTPAPIBackoffMaxDuration
	} else {
		b.duration = time.Duration(config.BackendHTTPAPIBackoffRate * float64(b.duration))
	}
}

func (b *backoff) sleep() {
	b.next()
	logger.Debugf("retrying the request in %s (number of failures: %d)", b.duration, b.fails)
	time.Sleep(b.duration)
}

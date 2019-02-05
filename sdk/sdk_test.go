package sdk

import (
	"net/http"
	"testing"
	"time"

	"github.com/sqreen/go-agent/agent"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func newFakeRequest() *http.Request {
	req, _ := http.NewRequest("GET", "https://sqreen.com", nil)
	return req
}

func TestSDK(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		testDisabledSDKCalls(t, nil)
	})

	t.Run("Track", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		eventId := testlib.RandString(2, 50)
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		event := ctx.TrackEvent(eventId)
		require.Equal(t, event.impl.GetName(), "track")

		t.Run("with user identifier", func(t *testing.T) {
			event.WithUserIdentifiers(idMap)
			args := event.impl.GetArgs()
			require.Equal(t, 2, len(args))
			require.Equal(t, args[0].(string), eventId)
			require.Equal(t, args[1].(*api.RequestRecord_Observed_SDKEvent_Options).GetUserIdentifiers().Value, agent.EventUserIdentifiersMap(idMap))
		})
	})

	t.Run("TrackAuth", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		success := true
		ctx.FromUser(idMap).TrackAuth(success)
	})

	t.Run("TrackSignup", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		ctx.ForUser(idMap).TrackSignup()
	})

	t.Run("TrackIdentify", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		ctx.ForUser(idMap).TrackIdentify()
	})
}

func testDisabledSDKCalls(t *testing.T, ctx *HTTPRequestRecord) {
	event := ctx.TrackEvent(testlib.RandString(0, 50))
	require.Nil(t, event)
	event = event.WithTimestamp(time.Now())
	require.Nil(t, event)
	event = event.WithProperties(nil)
	require.Nil(t, event)
	event = ctx.TrackEvent(testlib.RandString(0, 50))
	require.Nil(t, event)
	event = event.WithProperties(nil)
	require.Nil(t, event)
	event = event.WithTimestamp(time.Now())
	require.Nil(t, event)
	uid := EventUserIdentifiersMap{"uid": "uid"}
	ctx.ForUser(uid).TrackSignup().TrackAuth(true).TrackEvent("password.changed")
	ctx.ForUser(uid).TrackAuthSuccess()
	ctx.ForUser(uid).TrackAuthFailure()
	ctx.ForUser(uid).TrackSignup()
	ctx.Close()
}

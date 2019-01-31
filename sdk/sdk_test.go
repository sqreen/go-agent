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
		idMap := EventUserIdentifierMap{"uid": uid}
		event := ctx.TrackEvent(eventId)
		require.Equal(t, event.impl.GetName(), "track")

		t.Run("with user identifier", func(t *testing.T) {
			event.WithUserIdentifier(idMap)
			args := event.impl.GetArgs()
			require.Equal(t, 2, len(args))
			require.Equal(t, args[0].(string), eventId)
			require.Equal(t, args[1].(*api.RequestRecord_Observed_SDKEvent_Options).GetUserIdentifiers().Value, agent.EventUserIdentifierMap(idMap))
		})
	})

	t.Run("TrackAuth", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifierMap{"uid": uid}
		success := true
		ctx.TrackAuth(success, idMap)
	})

	t.Run("TrackSignup", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifierMap{"uid": uid}
		ctx.TrackSignup(idMap)
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
	uid := EventUserIdentifierMap{"uid": "uid"}
	ctx.TrackAuth(true, uid)
	ctx.TrackAuthSuccess(uid)
	ctx.TrackAuthFailure(uid)
	ctx.TrackSignup(uid)
	ctx.Close()
}

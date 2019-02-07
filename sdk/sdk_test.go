package sdk

import (
	"net/http"
	"testing"
	"time"

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
		})
	})

	t.Run("TrackAuth", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		success := true
		ctx.ForUser(idMap).TrackAuth(success)
	})

	t.Run("TrackSignup", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		ctx.ForUser(idMap).TrackSignup()
	})

	t.Run("Identify", func(t *testing.T) {
		ctx := NewHTTPRequestRecord(newFakeRequest())
		uid := testlib.RandString(2, 50)
		idMap := EventUserIdentifiersMap{"uid": uid}
		ctx.ForUser(idMap).TrackEvent("my.event")
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

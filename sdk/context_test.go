package sdk_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

type fakeRequest struct {
}

func (_ fakeRequest) StdRequest() *http.Request {
	req, _ := http.NewRequest("GET", "https://sqreen.com", nil)
	return req
}

func (_ fakeRequest) ClientIP() string {
	return ""
}

func TestSDK(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		testDisabledSDKCalls(t, nil)
	})

	t.Run("Track", func(t *testing.T) {
		ctx := sdk.NewHTTPRequestContext(fakeRequest{})
		eventId := testlib.RandString(2, 50)
		uid := testlib.RandString(2, 50)
		idMap := sdk.EventUserIdentifierMap{"uid": uid}
		event := ctx.Track(eventId)
		require.Equal(t, event.GetName(), "track")

		t.Run("with user identifier", func(t *testing.T) {
			event.WithUserIdentifier(idMap)
			args := event.GetArgs()
			require.Equal(t, 2, len(args))
			require.Equal(t, args[0].(string), eventId)
			require.Equal(t, args[1].(*api.RequestRecord_Observed_SDKEvent_Options).GetUserIdentifiers().Value, idMap)
		})
	})

	t.Run("TrackAuth", func(t *testing.T) {
		ctx := sdk.NewHTTPRequestContext(fakeRequest{})
		uid := testlib.RandString(2, 50)
		idMap := sdk.EventUserIdentifierMap{"uid": uid}
		success := true
		ctx.TrackAuth(success, idMap)
	})

	t.Run("TrackSignup", func(t *testing.T) {
		ctx := sdk.NewHTTPRequestContext(fakeRequest{})
		uid := testlib.RandString(2, 50)
		idMap := sdk.EventUserIdentifierMap{"uid": uid}
		ctx.TrackSignup(idMap)
	})
}

func testDisabledSDKCalls(t *testing.T, ctx *sdk.HTTPRequestContext) {
	event := ctx.Track(testlib.RandString(0, 50))
	require.Nil(t, event)
	event = event.WithTimestamp(time.Now())
	require.Nil(t, event)
	event = event.WithProperties(nil)
	require.Nil(t, event)
	event = ctx.Track(testlib.RandString(0, 50))
	require.Nil(t, event)
	event = event.WithProperties(nil)
	require.Nil(t, event)
	event = event.WithTimestamp(time.Now())
	require.Nil(t, event)
	uid := sdk.EventUserIdentifierMap{"uid": "uid"}
	ctx.TrackAuth(true, uid)
	ctx.TrackAuthSuccess(uid)
	ctx.TrackAuthFailure(uid)
	ctx.TrackSignup(uid)
	ctx.Close()
}

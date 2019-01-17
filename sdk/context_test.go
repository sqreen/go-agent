package sdk_test

import (
	"testing"
	"time"

	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestSDK(t *testing.T) {
	testDisabledSDKCalls(t, nil)
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
	ctx.Close()
}

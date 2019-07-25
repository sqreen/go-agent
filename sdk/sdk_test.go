// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	record := &sdk.HTTPRequestRecord{}

	t.Run("unset value", func(t *testing.T) {
		ctx := context.Background()
		got := sdk.FromContext(ctx)
		require.Nil(t, got)
	})

	t.Run("from a pointer key", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), sdk.HTTPRequestRecordContextKey, record)
		got := sdk.FromContext(ctx)
		require.NotNil(t, got)
	})

	t.Run("from a string key", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), sdk.HTTPRequestRecordContextKey.String, record)
		got := sdk.FromContext(ctx)
		require.NotNil(t, got)
	})
}

func TestGracefulStop(t *testing.T) {
	agent := &testlib.AgentMockup{}
	sdk.SetAgent(agent)
	defer agent.AssertExpectations(t)
	agent.ExpectGracefulStop().Once()
	sdk.GracefulStop()
}

func TestTrackEvent(t *testing.T) {
	agent := &testlib.AgentMockup{}
	defer agent.AssertExpectations(t)
	sdk.SetAgent(agent)
	record := &testlib.HTTPRequestRecordMockup{}
	defer record.AssertExpectations(t)
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()

	req := newTestRequest()
	sqReq := sdk.NewHTTPRequest(req)
	require.NotNil(t, sqReq)
	req = sqReq.Request()
	require.NotNil(t, req)

	sqreen := sdk.FromContext(req.Context())
	require.NotNil(t, sqreen)

	defer sqReq.Close()
	record.ExpectClose()

	eventID := testlib.RandString(2, 50)
	record.ExpectTrackEvent(eventID).Return(record).Once()

	sqEvent := sqreen.TrackEvent(eventID)
	require.NotNil(t, sqEvent)

	t.Run("with user identifiers", func(t *testing.T) {
		userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
		record.ExpectWithUserIdentifiers(userID).Once()
		sqEvent = sqEvent.WithUserIdentifiers(userID)
		require.NotNil(t, sqEvent)

		t.Run("chain with properties", func(t *testing.T) {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			record.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(t, sqEvent)
		})

		t.Run("chain with timestamp", func(t *testing.T) {
			timestamp := time.Now()
			record.ExpectWithTimestamp(timestamp).Once()
			sqEvent = sqEvent.WithTimestamp(timestamp)
			require.NotNil(t, sqEvent)
		})
	})

	t.Run("with properties", func(t *testing.T) {
		require := require.New(t)
		props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
		record.ExpectWithProperties(props).Once()
		sqEvent = sqEvent.WithProperties(props)
		require.NotNil(sqEvent)

		t.Run("chain with user identifiers", func(t *testing.T) {
			userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			record.ExpectWithUserIdentifiers(userID).Once()
			sqEvent = sqEvent.WithUserIdentifiers(userID)
			require.NotNil(t, sqEvent)
		})

		t.Run("chain with timestamp", func(t *testing.T) {
			timestamp := time.Now()
			record.ExpectWithTimestamp(timestamp).Once()
			sqEvent = sqEvent.WithTimestamp(timestamp)
			require.NotNil(t, sqEvent)
		})
	})

	t.Run("with timestamp", func(t *testing.T) {
		require := require.New(t)
		timestamp := time.Now()
		record.ExpectWithTimestamp(timestamp).Once()
		sqEvent = sqEvent.WithTimestamp(timestamp)
		require.NotNil(sqEvent)

		t.Run("chain with user identifiers", func(t *testing.T) {
			userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			record.ExpectWithUserIdentifiers(userID).Once()
			sqEvent = sqEvent.WithUserIdentifiers(userID)
			require.NotNil(sqEvent)
		})

		t.Run("chain with properties", func(t *testing.T) {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			record.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(sqEvent)
		})
	})
}

func TestForUser(t *testing.T) {
	agent := &testlib.AgentMockup{}
	defer agent.AssertExpectations(t)
	sdk.SetAgent(agent)
	record := &testlib.HTTPRequestRecordMockup{}
	defer record.AssertExpectations(t)
	req := newTestRequest()
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()

	sqReq := sdk.NewHTTPRequest(req)
	require.NotNil(t, sqReq)
	req = sqReq.Request()
	require.NotNil(t, req)

	sqreen := sdk.FromContext(req.Context())
	require.NotNil(t, sqreen)

	defer sqReq.Close()
	record.ExpectClose()

	userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}

	sqUser := sqreen.ForUser(userID)
	require.NotNil(t, sqUser)

	t.Run("TrackAuth", func(t *testing.T) {
		record.ExpectTrackAuth(userID, true).Once()
		sqUser = sqUser.TrackAuth(true)
		require.NotNil(t, sqUser)

		record.ExpectTrackAuth(userID, true).Once()
		sqUser = sqUser.TrackAuthSuccess()
		require.NotNil(t, sqUser)

		record.ExpectTrackAuth(userID, false).Once()
		sqUser = sqUser.TrackAuth(false)
		require.NotNil(t, sqUser)

		record.ExpectTrackAuth(userID, false).Once()
		sqUser = sqUser.TrackAuthFailure()
		require.NotNil(t, sqUser)
	})

	t.Run("TrackSignup", func(t *testing.T) {
		record.ExpectTrackSignup(userID).Once()
		sqUser = sqUser.TrackSignup()
		require.NotNil(t, sqUser)
	})

	t.Run("Identify", func(t *testing.T) {
		record.ExpectIdentify(userID).Once()
		sqUser = sqUser.Identify()
		require.NotNil(t, sqUser)
	})

	t.Run("MatchSecurityResponse", func(t *testing.T) {
		t.Run("without security response", func(t *testing.T) {
			record.ExpectUserSecurityResponse().Return(http.Handler(nil)).Once()
			match, err := sqUser.MatchSecurityResponse()
			require.NoError(t, err)
			require.False(t, match)
		})

		t.Run("with security response", func(t *testing.T) {
			handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
			record.ExpectUserSecurityResponse().Return(handler).Once()
			match, err := sqUser.MatchSecurityResponse()
			require.Error(t, err)
			require.NotEmpty(t, err.Error())
			require.True(t, match)
		})
	})

	t.Run("TrackEvent", func(t *testing.T) {
		eventID := testlib.RandString(2, 50)
		record.ExpectTrackEvent(eventID).Once()
		record.ExpectWithUserIdentifiers(userID).Once()
		sqEvent := sqUser.TrackEvent(eventID)
		require.NotNil(t, sqEvent)

		t.Run("with properties", func(t *testing.T) {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			record.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(t, sqEvent)

			t.Run("chain with timestamp", func(t *testing.T) {
				timestamp := time.Now()
				record.ExpectWithTimestamp(timestamp).Once()
				sqEvent = sqEvent.WithTimestamp(timestamp)
				require.NotNil(t, sqEvent)
			})
		})

		t.Run("with timestamp", func(t *testing.T) {
			timestamp := time.Now()
			record.ExpectWithTimestamp(timestamp).Once()
			sqEvent = sqEvent.WithTimestamp(timestamp)
			require.NotNil(t, sqEvent)

			t.Run("chain with properties", func(t *testing.T) {
				props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
				record.ExpectWithProperties(props).Once()
				sqEvent = sqEvent.WithProperties(props)
				require.NotNil(t, sqEvent)
			})
		})
	})
}

type whitelistedRecord struct{}

func (r whitelistedRecord) NewCustomEvent(event string) types.CustomEvent { return r }
func (whitelistedRecord) NewUserSignup(id map[string]string)              {}
func (whitelistedRecord) NewUserAuth(id map[string]string, success bool)  {}
func (whitelistedRecord) Identify(id map[string]string)                   {}
func (whitelistedRecord) SecurityResponse() http.Handler                  { return nil }
func (whitelistedRecord) UserSecurityResponse() http.Handler              { return nil }
func (whitelistedRecord) Close()                                          {}
func (whitelistedRecord) WithTimestamp(t time.Time)                       {}
func (whitelistedRecord) WithProperties(props types.EventProperties)      {}
func (whitelistedRecord) WithUserIdentifiers(id map[string]string)        {}

// Using the SDK shouldn't fail when disabled.
func TestDisabled(t *testing.T) {
	useTheSDK := func(t *testing.T, sqreen *sdk.HTTPRequestRecord) func() {
		return func() {
			event := sqreen.TrackEvent(testlib.RandString(0, 50))
			event = event.WithTimestamp(time.Now())
			userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 30): testlib.RandString(2, 30)}
			event = event.WithUserIdentifiers(userID)
			props := sdk.EventPropertyMap{testlib.RandString(2, 30): testlib.RandString(2, 30)}
			event = event.WithProperties(props)
			uid := sdk.EventUserIdentifiersMap{testlib.RandString(2, 30): testlib.RandString(2, 30)}
			sqUser := sqreen.ForUser(uid)
			sqUser = sqUser.TrackSignup()
			sqUser = sqUser.TrackAuth(true)
			sqUser = sqUser.TrackAuthSuccess()
			sqUser = sqUser.TrackAuthFailure()
			sqUser = sqUser.Identify()
			match, err := sqUser.MatchSecurityResponse()
			require.False(t, match)
			require.NoError(t, err)
			sqUserEvent := sqUser.TrackEvent(testlib.RandString(0, 50))
			sqUserEvent = sqUserEvent.WithProperties(props)
			sqUserEvent = sqUserEvent.WithTimestamp(time.Now())
			sqreen.Close()
		}
	}

	t.Run("with a disabled agent", func(t *testing.T) {
		sdk.SetAgent(nil)
		// When getting the SDK context out of a bare Go context, ie. without sqreen's
		// middleware modifications.
		sqreen := sdk.FromContext(context.Background())
		require.NotPanics(t, useTheSDK(t, sqreen))

		// When not even following the proper SDK usage (middlewares, etc.).
		require.NotPanics(t, useTheSDK(t, nil))

		// When getting the SDK context out of the request wrapper.
		req := sdk.NewHTTPRequest(newTestRequest())
		record := req.Record()
		require.NotPanics(t, useTheSDK(t, record))

		// Other methods
		req.SecurityResponse()
		req.UserSecurityResponse()
		req.Close()
		sdk.GracefulStop()
	})
}

func TestEventPropertyMap(t *testing.T) {
	key := testlib.RandString(1, 100)
	value := testlib.RandString(1, 100)
	props := sdk.EventPropertyMap{
		key: value,
	}
	buf, err := props.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, string(buf), fmt.Sprintf(`{"%s":"%s"}`, key, value))
}

func newTestRequest() *http.Request {
	req := httptest.NewRequest("GET", "https://sqreen.com", nil)
	return req
}

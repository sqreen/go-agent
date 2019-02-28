package sdk_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SDKTestSuite struct {
	suite.Suite
	agent *testlib.AgentMockup
}

func (suite *SDKTestSuite) SetupTest() {
	suite.agent = &testlib.AgentMockup{}
	sdk.SetAgent(suite.agent)
}

func (suite *SDKTestSuite) TearDownTest() {
	suite.agent.AssertExpectations(suite.T())
}

func (suite *SDKTestSuite) TestFromContext() {
	require := require.New(suite.T())

	req := newTestRequest()
	suite.agent.ExpectNewRequestRecord(req).Once()

	sqreen := sdk.NewHTTPRequestRecord(req)
	require.NotNil(sqreen)

	ctx := context.WithValue(context.Background(), sdk.HTTPRequestRecordContextKey, sqreen)

	got := sdk.FromContext(ctx)
	require.Equal(got, sqreen)
}

func (suite *SDKTestSuite) TestGracefulStop() {
	suite.agent.ExpectGracefulStop().Once()
	sdk.GracefulStop()
}

func (suite *SDKTestSuite) TestSecurityAction() {
	req := newTestRequest()
	action := &testlib.SecurityActionMockup{}
	suite.agent.ExpectSecurityAction(req).Once().Return(action)
	got := sdk.SecurityAction(req)
	require.Equal(suite.T(), got, action)
}

func (suite *SDKTestSuite) TestTrackEvent() {
	require := require.New(suite.T())

	req := newTestRequest()
	suite.agent.ExpectNewRequestRecord(req).Once()

	sqreen := sdk.NewHTTPRequestRecord(req)
	require.NotNil(sqreen)

	eventID := testlib.RandString(2, 50)
	suite.agent.ExpectTrackEvent(eventID).Once()
	sqEvent := sqreen.TrackEvent(eventID)
	require.NotNil(sqEvent)

	suite.Run("with user identifiers", func() {
		userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
		suite.agent.ExpectWithUserIdentifiers(userID).Once()
		sqEvent = sqEvent.WithUserIdentifiers(userID)
		require.NotNil(sqEvent)

		suite.Run("chain with properties", func() {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			suite.agent.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(sqEvent)
		})

		suite.Run("chain with timestamp", func() {
			t := time.Now()
			suite.agent.ExpectWithTimestamp(t).Once()
			sqEvent = sqEvent.WithTimestamp(t)
			require.NotNil(sqEvent)
		})
	})

	suite.Run("with properties", func() {
		props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
		suite.agent.ExpectWithProperties(props).Once()
		sqEvent = sqEvent.WithProperties(props)
		require.NotNil(sqEvent)

		suite.Run("chain with user identifiers", func() {
			userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			suite.agent.ExpectWithUserIdentifiers(userID).Once()
			sqEvent = sqEvent.WithUserIdentifiers(userID)
			require.NotNil(sqEvent)
		})

		suite.Run("chain with timestamp", func() {
			t := time.Now()
			suite.agent.ExpectWithTimestamp(t).Once()
			sqEvent = sqEvent.WithTimestamp(t)
			require.NotNil(sqEvent)
		})
	})

	suite.Run("with timestamp", func() {
		t := time.Now()
		suite.agent.ExpectWithTimestamp(t).Once()
		sqEvent = sqEvent.WithTimestamp(t)
		require.NotNil(sqEvent)

		suite.Run("chain with user identifiers", func() {
			userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			suite.agent.ExpectWithUserIdentifiers(userID).Once()
			sqEvent = sqEvent.WithUserIdentifiers(userID)
			require.NotNil(sqEvent)
		})

		suite.Run("chain with properties", func() {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			suite.agent.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(sqEvent)
		})
	})

}

func (suite *SDKTestSuite) TestForUser() {
	require := require.New(suite.T())

	req := newTestRequest()
	suite.agent.ExpectNewRequestRecord(req)
	sqreen := sdk.NewHTTPRequestRecord(req)
	require.NotNil(sqreen)

	userID := sdk.EventUserIdentifiersMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}

	sqUser := sqreen.ForUser(userID)
	require.NotNil(sqUser)

	suite.Run("TrackAuth", func() {
		suite.agent.ExpectTrackAuth(userID, true).Once()
		sqUser = sqUser.TrackAuth(true)
		require.NotNil(sqUser)

		suite.agent.ExpectTrackAuth(userID, false).Once()
		sqUser = sqUser.TrackAuth(false)
		require.NotNil(sqUser)
	})

	suite.Run("TrackSignup", func() {
		suite.agent.ExpectTrackSignup(userID).Once()
		sqUser = sqUser.TrackSignup()
		require.NotNil(sqUser)
	})

	suite.Run("Identfy", func() {
		suite.agent.ExpectIdentify(userID).Once()
		sqUser = sqUser.Identify()
		require.NotNil(sqUser)
	})

	suite.Run("TrackEvent", func() {
		eventID := testlib.RandString(2, 50)
		suite.agent.ExpectTrackEvent(eventID).Once()
		suite.agent.ExpectIdentify(userID).Once()
		sqEvent := sqUser.TrackEvent(eventID)
		require.NotNil(sqEvent)

		suite.Run("with properties", func() {
			props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
			suite.agent.ExpectWithProperties(props).Once()
			sqEvent = sqEvent.WithProperties(props)
			require.NotNil(sqEvent)

			suite.Run("chain with timestamp", func() {
				t := time.Now()
				suite.agent.ExpectWithTimestamp(t).Once()
				sqEvent = sqEvent.WithTimestamp(t)
				require.NotNil(sqEvent)
			})
		})

		suite.Run("with timestamp", func() {
			t := time.Now()
			suite.agent.ExpectWithTimestamp(t).Once()
			sqEvent = sqEvent.WithTimestamp(t)
			require.NotNil(sqEvent)

			suite.Run("chain with properties", func() {
				props := sdk.EventPropertyMap{testlib.RandString(2, 50): testlib.RandString(2, 50)}
				suite.agent.ExpectWithProperties(props).Once()
				sqEvent = sqEvent.WithProperties(props)
				require.NotNil(sqEvent)
			})
		})
	})
}

func TestDisabled(t *testing.T) {
	require := require.New(t)
	sdk.SetAgent(nil)

	useTheSDK := func(sqreen *sdk.HTTPRequestRecord) func() {
		return func() {
			sdk.SecurityAction(nil)
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
			sqUserEvent := sqUser.TrackEvent(testlib.RandString(0, 50))
			sqUserEvent = sqUserEvent.WithProperties(props)
			sqUserEvent = sqUserEvent.WithTimestamp(time.Now())
			sqreen.Close()
		}
	}

	// Using the SDK shouldn't fail.

	// When getting the SDK context out of a bare Go context, ie. without sqreen's
	// middleware modifications.
	sqreen := sdk.FromContext(context.Background())
	require.NotPanics(useTheSDK(sqreen))

	// When creating the request record ourselves.
	sqreen = sdk.NewHTTPRequestRecord(newTestRequest())
	require.NotPanics(useTheSDK(sqreen))

	// When not even following the SDK requirements.
	require.NotPanics(useTheSDK(nil))
}

func TestSDK(t *testing.T) {
	suite.Run(t, new(SDKTestSuite))
}

func newTestRequest() *http.Request {
	req, _ := http.NewRequest("GET", "https://sqreen.com", nil)
	return req
}

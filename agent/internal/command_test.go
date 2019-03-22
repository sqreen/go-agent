package internal_test

import (
	"errors"
	"testing"

	"github.com/sqreen/go-agent/agent/internal"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCommandManager(t *testing.T) {
	var agent agentMockup
	logger := plog.NewLogger("test", nil)
	mng := internal.NewCommandManager(&agent, logger)
	require.NotNil(t, mng)

	t.Run("nil command list", func(t *testing.T) {
		results := mng.Do(nil)
		require.Nil(t, results)
	})

	t.Run("empty command list", func(t *testing.T) {
		results := mng.Do([]api.CommandRequest{})
		require.Nil(t, results)
	})

	t.Run("unknown command", func(t *testing.T) {
		uuid := testlib.RandString(1, 126)
		results := mng.Do([]api.CommandRequest{
			{
				Uuid: uuid,
				Name: testlib.RandString(1, 50),
			},
		})
		require.False(t, results[uuid].Status)
		agent.AssertExpectations(t)
	})

	testCases := []struct {
		Command           string
		AgentExpectedCall func() *mock.Call
	}{
		{
			Command:           "instrumentation_enable",
			AgentExpectedCall: agent.ExpectInstrumentationEnable,
		},
		{
			Command:           "instrumentation_remove",
			AgentExpectedCall: agent.ExpectInstrumentationDisable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Command, func(t *testing.T) {
			t.Run("without errors", func(t *testing.T) {
				agent.Reset()

				tc.AgentExpectedCall().Return(nil).Once()
				uuid := testlib.RandString(1, 126)
				results := mng.Do([]api.CommandRequest{
					{
						Uuid: uuid,
						Name: tc.Command,
					},
				})
				require.Equal(t, results, map[string]api.CommandResult{
					uuid: api.CommandResult{
						Status: true,
						Output: "",
					},
				})
				agent.AssertExpectations(t)
			})

			t.Run("with error", func(t *testing.T) {
				agent.Reset()

				errorMsg := testlib.RandString(1, 126)
				tc.AgentExpectedCall().Return(errors.New(errorMsg)).Once()
				uuid := testlib.RandString(1, 126)
				results := mng.Do([]api.CommandRequest{
					{
						Uuid: uuid,
						Name: tc.Command,
					},
				})
				require.Equal(t, results, map[string]api.CommandResult{
					uuid: api.CommandResult{
						Status: false,
						Output: errorMsg,
					},
				})
				agent.AssertExpectations(t)
			})
		})
	}

	t.Run("multiple commands", func(t *testing.T) {
		agent.Reset()

		var (
			commands        []api.CommandRequest
			expectedResults = make(map[string]api.CommandResult)
		)

		// Generate the list of commands and the expected results
		for _, tc := range testCases {
			uuid := testlib.RandString(1, 126)

			commands = append(commands, api.CommandRequest{
				Uuid: uuid,
				Name: tc.Command,
			})

			expectedResults[uuid] = api.CommandResult{
				Status: true,
				Output: "",
			}

			tc.AgentExpectedCall().Return(nil).Once()
		}

		// Also include wrong commands
		for n := 0; n <= int(testlib.RandUint32(1)); n++ {
			uuid := testlib.RandString(1, 126)

			commands = append(commands, api.CommandRequest{
				Uuid: uuid,
				Name: testlib.RandString(1, 50),
			})

			expectedResults[uuid] = api.CommandResult{
				Status: false,
				Output: config.ErrorMessage_UnsupportedCommand,
			}
		}

		results := mng.Do(commands)
		require.Equal(t, results, expectedResults)
	})
}

type agentMockup struct {
	mock.Mock
}

func (a *agentMockup) Reset() {
	a.Mock = mock.Mock{}
}

func (a *agentMockup) InstrumentationEnable() error {
	ret := a.Called()
	return ret.Error(0)
}

func (a *agentMockup) InstrumentationDisable() error {
	ret := a.Called()
	return ret.Error(0)
}

func (a *agentMockup) ExpectInstrumentationEnable() *mock.Call {
	return a.On("InstrumentationEnable")
}

func (a *agentMockup) ExpectInstrumentationDisable() *mock.Call {
	return a.On("InstrumentationDisable")
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal_test

import (
	"encoding/json"
	"errors"
	"os"
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
	logger := plog.NewLogger(plog.Debug, os.Stderr, 0)
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
		AgentExpectedCall func(args ...interface{}) *mock.Call
		ExpectedArgs      []interface{}
		Args              []json.RawMessage
		BadArgs           [][]json.RawMessage
	}{
		{
			Command:           "instrumentation_enable",
			AgentExpectedCall: agent.ExpectInstrumentationEnable,
		},
		{
			Command:           "instrumentation_remove",
			AgentExpectedCall: agent.ExpectInstrumentationDisable,
		},
		{
			Command:           "actions_reload",
			AgentExpectedCall: agent.ExpectActionsReload,
		},
		{
			Command:           "ips_whitelist",
			Args:              []json.RawMessage{json.RawMessage(`["a","b","c"]`)},
			ExpectedArgs:      []interface{}{[]string{"a", "b", "c"}},
			AgentExpectedCall: agent.ExpectSetCIDRWhitelist,
			BadArgs: [][]json.RawMessage{
				{json.RawMessage(`"wrong type"`)},
				{json.RawMessage(`[1,2,3]`)},
				{json.RawMessage(`["a","b","c"]`), json.RawMessage(`"wrong count"`)},
				{json.RawMessage(`["a", "b", "c"]`), json.RawMessage(`["a", "b", "c"]`)},
			},
		},
		{
			Command:           "rules_reload",
			AgentExpectedCall: agent.ExpectRulesReload,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Command, func(t *testing.T) {
			t.Run("without errors", func(t *testing.T) {
				agent.Reset()

				tc.AgentExpectedCall(tc.ExpectedArgs...).Return(nil).Once()
				uuid := testlib.RandString(1, 126)
				results := mng.Do([]api.CommandRequest{
					{
						Uuid:   uuid,
						Name:   tc.Command,
						Params: tc.Args,
					},
				})
				require.Equal(
					t,
					map[string]api.CommandResult{
						uuid: api.CommandResult{
							Status: true,
							Output: "",
						},
					},
					results,
				)
				agent.AssertExpectations(t)
			})

			t.Run("with errors", func(t *testing.T) {
				agent.Reset()

				errorMsg := testlib.RandString(1, 126)
				tc.AgentExpectedCall(tc.ExpectedArgs...).Return(errors.New(errorMsg)).Once()
				uuid := testlib.RandString(1, 126)
				results := mng.Do([]api.CommandRequest{
					{
						Uuid:   uuid,
						Name:   tc.Command,
						Params: tc.Args,
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

			if len(tc.BadArgs) > 0 {
				t.Run("with args errors", func(t *testing.T) {
					for _, args := range tc.BadArgs {
						args := args // new scope
						agent.Reset()
						// No agent calls are expected

						uuid := testlib.RandString(1, 126)
						results := mng.Do([]api.CommandRequest{
							{
								Uuid:   uuid,
								Name:   tc.Command,
								Params: args,
							},
						})
						require.False(t, results[uuid].Status)
						agent.AssertExpectations(t)
					}
				})
			}
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
				Uuid:   uuid,
				Name:   tc.Command,
				Params: tc.Args,
			})

			expectedResults[uuid] = api.CommandResult{
				Status: true,
				Output: "",
			}

			tc.AgentExpectedCall(tc.ExpectedArgs...).Return(nil).Once()
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
		agent.AssertExpectations(t)
	})

	t.Run("repeated commands", func(t *testing.T) {
		agent.Reset()

		var (
			commands        []api.CommandRequest
			expectedResults = make(map[string]api.CommandResult)
		)

		// Generate the list of commands and the expected results
		for _, tc := range testCases {
			uuid := testlib.RandString(1, 126)
			uuid2 := testlib.RandString(1, 126)

			commands = append(commands, api.CommandRequest{
				Uuid:   uuid,
				Name:   tc.Command,
				Params: tc.Args,
			})

			commands = append(commands, api.CommandRequest{
				Uuid:   uuid2,
				Name:   tc.Command,
				Params: tc.Args,
			})

			expectedResults[uuid] = api.CommandResult{
				Status: true,
				Output: "",
			}

			expectedResults[uuid2] = api.CommandResult{
				Status: true,
				Output: "",
			}

			tc.AgentExpectedCall(tc.ExpectedArgs...).Return(nil).Once() // Checks command are performed just once
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
		agent.AssertExpectations(t)
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

func (a *agentMockup) ActionsReload() error {
	ret := a.Called()
	return ret.Error(0)
}

func (a *agentMockup) SetCIDRWhitelist(cidrs []string) error {
	ret := a.Called(cidrs)
	return ret.Error(0)
}

func (a *agentMockup) RulesReload() error {
	ret := a.Called()
	return ret.Error(0)
}

func (a *agentMockup) ExpectInstrumentationEnable(...interface{}) *mock.Call {
	return a.On("InstrumentationEnable")
}

func (a *agentMockup) ExpectInstrumentationDisable(...interface{}) *mock.Call {
	return a.On("InstrumentationDisable")
}

func (a *agentMockup) ExpectActionsReload(...interface{}) *mock.Call {
	return a.On("ActionsReload")
}

func (a *agentMockup) ExpectSetCIDRWhitelist(args ...interface{}) *mock.Call {
	return a.On("SetCIDRWhitelist", args...)
}

func (a *agentMockup) ExpectRulesReload(...interface{}) *mock.Call {
	return a.On("RulesReload")
}

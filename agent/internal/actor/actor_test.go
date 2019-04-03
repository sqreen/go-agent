package actor_test

import (
	"math/rand"
	"net"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/agent/internal/actor"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

var (
	logger = plog.NewLogger("test", nil)
	fuzzer = fuzz.New().NilChance(0)
)

func TestStore(t *testing.T) {
	t.Run("SetActions", func(t *testing.T) {
		t.Run("The number of actions should be limited", func(t *testing.T) {
			t.Skip("too long - make it configurable through the package API")
			actors := actor.NewStore(logger)
			var (
				actions []api.ActionsPackResponse_Action
				err     error
			)
			for err == nil {
				actions = append(actions, *NewBlockIPAction("1.2.3.5/27"))
				err = actors.SetActions(actions)
			}
			require.Error(t, err)
		})

		t.Run("Malformed action list", func(t *testing.T) {
			t.Parallel()
			actors := actor.NewStore(logger)
			err := actors.SetActions([]api.ActionsPackResponse_Action{
				{
					Action:   "block_ip",
					Duration: 0,
					Parameters: api.ActionsPackResponse_Action_Params{
						IpCidr: nil,
					},
				},
			})
			require.Error(t, err)
		})

		t.Run("Nil action list", func(t *testing.T) {
			t.Parallel()
			actors := actor.NewStore(logger)
			err := actors.SetActions(nil)
			require.NoError(t, err)
		})

		t.Run("Empty action list", func(t *testing.T) {
			t.Parallel()
			actors := actor.NewStore(logger)
			err := actors.SetActions([]api.ActionsPackResponse_Action{})
			require.NoError(t, err)
		})

		tests := []struct {
			name string
			// Map of actions with their list of CIDRs to test against them. It allows
			// to check that successful calls to FindIP() returns the correct
			// action. SetActions() is expected to fail when all the values are nil.
			actions map[*api.ActionsPackResponse_Action][]net.IP
			// List of IPs that should fail. So we don't need to keep the link to
			// the original action.
			findIPFailure []net.IP
		}{
			{
				name:          "Empty store",
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(80, 64, 3, 221), RandIPv4(), RandIPv4(), RandIPv4()},
			},

			{
				name: "Number of bits too high",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/35"): nil,
				},
			},

			{
				name: "Malformed",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4:80"): nil,
				},
			},

			{
				name: "Malformed",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2..3.4/0"): nil,
				},
			},

			{
				name: "Malformed",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction(""): nil,
				},
			},

			{
				name: "Malformed",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction(testlib.RandString(2, 10)): nil,
				},
			},

			{
				name: "1.2.3.4/32",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4"): []net.IP{{1, 2, 3, 4}},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3)},
			},

			{
				name: "1.2.3.4/32",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/32"): []net.IP{net.IPv4(1, 2, 3, 4)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3)},
			},

			{
				name: "1.2.3.4/32",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/32", "1.2.3.4"): []net.IP{net.IPv4(1, 2, 3, 4)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3)},
			},

			{
				name: "Overalpping Networks",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/16", "1.2.3.4/32", "1.2.3.4/24"): []net.IP{
						net.IPv4(1, 2, 3, 4),
						net.IPv4(1, 2, 3, 5),
						net.IPv4(1, 2, 3, 120),
						net.IPv4(1, 2, 3, 255),
						net.IPv4(1, 2, 4, 0),
						net.IPv4(1, 2, 4, 33),
						net.IPv4(1, 2, 4, 255),
						net.IPv4(1, 2, 0, 0),
						net.IPv4(1, 2, 0, 1),
						net.IPv4(1, 2, 255, 255),
					},
				},
				findIPFailure: []net.IP{net.IPv4(1, 1, 255, 255), net.IPv4(1, 3, 0, 0)},
			},

			{
				name: "Overlapping Networks",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/32"): []net.IP{net.IPv4(1, 2, 3, 4)},
					NewBlockIPAction("1.2.3.4/24"): []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 120), net.IPv4(1, 2, 3, 255)},
					NewBlockIPAction("1.2.3.4/16"): []net.IP{
						net.IPv4(1, 2, 4, 0),
						net.IPv4(1, 2, 4, 33),
						net.IPv4(1, 2, 4, 255),
						net.IPv4(1, 2, 0, 0),
						net.IPv4(1, 2, 0, 1),
						net.IPv4(1, 2, 255, 255),
					},
				},
				findIPFailure: []net.IP{net.IPv4(1, 1, 255, 255), net.IPv4(1, 3, 0, 0)},
			},

			{
				name: "Subsequent CIDRs in a single action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4", "1.2.3.5", "1.2.3.3"): []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3), net.IPv4(1, 2, 3, 4)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6)},
			},

			{
				name: "Subsequest CIDRs per action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.5"): []net.IP{net.IPv4(1, 2, 3, 5)},
					NewBlockIPAction("1.2.3.4"): []net.IP{net.IPv4(1, 2, 3, 4)},
					NewBlockIPAction("1.2.3.3"): []net.IP{net.IPv4(1, 2, 3, 3)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6)},
			},
		}

		for _, tc := range tests {
			tc := tc // new scope for the following closure
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				t.Logf("%+v", tc)

				// Create the list of actions from the map keys.
				var actions []api.ActionsPackResponse_Action
				var setActionsShouldFail bool
				if tc.actions == nil {
					setActionsShouldFail = false
				} else {
					setActionsShouldFail = true
					for action := range tc.actions {
						actions = append(actions, *action)
						if len(tc.actions[action]) > 0 {
							// The convention here is that `SetActions()` calls that are
							// expected to fail don't have a list of CIDR string to test
							// against `FindIP()`.
							setActionsShouldFail = false
						}
					}
				}

				// Create an actor store.
				actors := actor.NewStore(logger)
				// Set its actions and check its returned error according to the test
				// case.
				err := actors.SetActions(actions)
				if setActionsShouldFail {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				// Test the store in parallel as it is meant to be thread-safe.
				t.Run("FindIP", func(t *testing.T) {
					for action, ips := range tc.actions {
						action := action // new scope for the following closure

						// Accesses that should be successful.
						for _, ip := range ips {
							ip := ip // new scope for the following closure
							t.Run(ip.String(), func(t *testing.T) {
								t.Parallel()
								t.Logf("%v", tc)
								t.Logf("FindIP(%s)", ip)
								got, exists, err := actors.FindIP(ip)
								require.NoError(t, err)
								require.True(t, exists)
								require.Equal(t, got.ActionID(), action.ActionId)
							})
						}
					}

					// Accesses that should fail
					for _, ip := range tc.findIPFailure {
						ip := ip // new scope for the following closure
						t.Run(ip.String(), func(t *testing.T) {
							t.Parallel()
							t.Logf("FindIP(%s)", ip)
							_, exists, err := actors.FindIP(ip)
							require.NoError(t, err)
							require.False(t, exists)
						})
					}
				})
			})
		}

		t.Run("Timed actions", func(t *testing.T) {
			t.Parallel()
			// Create an actor store.
			actors := actor.NewStore(logger)
			// Set its actions and check its returned error according to the test
			// case.
			actions := []api.ActionsPackResponse_Action{
				*NewTimedBlockIPAction(1*time.Second, "1.2.3.4/32"), // Enough time to perform the test
				*NewBlockIPAction("1.2.3.4/24"),
				*NewTimedBlockIPAction(1*time.Second, "1.2.3.4/16"), // Enough time to perform the test
			}
			err := actors.SetActions(actions)
			require.NoError(t, err)

			// Accesses that should be successful.

			// Start with timed actions first so that they shouldn't have expired yet
			timed := actions[2]
			for _, ip := range []net.IP{
				net.IPv4(1, 2, 4, 0),
				net.IPv4(1, 2, 4, 33),
				net.IPv4(1, 2, 4, 255),
				net.IPv4(1, 2, 0, 0),
				net.IPv4(1, 2, 0, 1),
				net.IPv4(1, 2, 255, 255),
			} {
				got, exists, err := actors.FindIP(ip)
				require.NoError(t, err)
				require.True(t, exists)
				require.Equal(t, got.ActionID(), timed.ActionId)
			}

			// Test against action #0
			got, exists, err := actors.FindIP(net.IPv4(1, 2, 3, 4))
			require.NoError(t, err)
			require.True(t, exists)
			// The most specific match is returned, which is /32 while it hasn't
			// expired.
			require.Equal(t, got.ActionID(), actions[0].ActionId)

			// Wait for the timed duration to make sure it has expired
			time.Sleep(timed.Duration)
			// Tests should be false now
			for _, ip := range []net.IP{
				net.IPv4(1, 2, 4, 0),
				net.IPv4(1, 2, 4, 33),
				net.IPv4(1, 2, 4, 255),
				net.IPv4(1, 2, 0, 0),
				net.IPv4(1, 2, 0, 1),
				net.IPv4(1, 2, 255, 255),
			} {
				_, exists, err := actors.FindIP(ip)
				require.NoError(t, err)
				require.False(t, exists) // The IP does not exist
			}

			got, exists, err = actors.FindIP(net.IPv4(1, 2, 3, 4))
			require.NoError(t, err)
			require.True(t, exists)
			// The most specific match (/32) has now expired, so the /24 action should
			// now be returned.
			require.Equal(t, got.ActionID(), actions[1].ActionId)
		})
	})
}

func NewBlockIPAction(CIDRs ...string) *api.ActionsPackResponse_Action {
	action := &api.ActionsPackResponse_Action{
		Action: "block_ip",
		Parameters: api.ActionsPackResponse_Action_Params{
			IpCidr: CIDRs,
		},
	}
	fuzzer.Fuzz(&action.ActionId)
	fuzzer.Fuzz(&action.SendResponse)
	return action
}

func NewTimedBlockIPAction(d time.Duration, CIDRs ...string) *api.ActionsPackResponse_Action {
	action := &api.ActionsPackResponse_Action{
		Action:   "block_ip",
		Duration: d,
		Parameters: api.ActionsPackResponse_Action_Params{
			IpCidr: CIDRs,
		},
	}
	fuzzer.Fuzz(&action.ActionId)
	fuzzer.Fuzz(&action.SendResponse)
	return action
}

func RandIPv4() net.IP {
	return net.IPv4(byte(rand.Uint32()), byte(rand.Uint32()), byte(rand.Uint32()), byte(rand.Uint32()))
}

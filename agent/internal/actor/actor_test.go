package actor_test

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
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
	logger = plog.NewLogger(plog.Debug, os.Stderr, 0)
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
	})

	t.Run("User", func(t *testing.T) {
		tests := []struct {
			name string
			// List of actions with their list of user-IDs to test against them.
			// It allows to check that successful calls to FindUser() returns the
			// correct action. SetActions() is expected to fail when all the values
			// are nil.
			actions []*api.ActionsPackResponse_Action
		}{
			{
				name: "one action and one user",
				actions: []*api.ActionsPackResponse_Action{
					NewBlockUserAction(map[string]string{"uid": "oh my uid"}),
				},
			},

			{
				name: "one action and one user with multiple fields",
				actions: []*api.ActionsPackResponse_Action{
					NewBlockUserAction(map[string]string{"uid": "oh my uid"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid", "field2": "value 21"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid", "field2": "value 21", "field3": "value 31"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid", "field2": "value 21", "field3": "value 32"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid", "field2": "value 22", "field3": "value 33"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid", "field2": "value 23", "field3": "value 33"}),
				},
			},

			{
				name: "one action and several users",
				actions: []*api.ActionsPackResponse_Action{
					NewBlockUserAction(
						map[string]string{"uid": "oh my uid"},
						map[string]string{"uid": "oh my uid 2"},
						map[string]string{"uid": "oh my uid 3"},
						map[string]string{"uid": "oh my uid 4"},
						map[string]string{"uid": "oh my uid 5"},
						map[string]string{"uid": "oh my uid 6"},
						map[string]string{"uid": "oh my uid 7"},
						RandUser(),
						RandUser(),
						RandUser(),
						RandUser(),
						RandUser(),
					),
				},
			},

			{
				name: "several actions with one user",
				actions: []*api.ActionsPackResponse_Action{
					NewBlockUserAction(map[string]string{"uid": "oh my uid"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid 2"}),
					NewBlockUserAction(map[string]string{"uid": "oh my uid 3"}),
					NewBlockUserAction(RandUser()),
					NewBlockUserAction(RandUser()),
					NewBlockUserAction(RandUser()),
					NewBlockUserAction(RandUser()),
				},
			},
		}

		for _, tc := range tests {
			tc := tc // new scope for the following closure
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				// Create the list of actions from the map keys.
				var (
					actions []api.ActionsPackResponse_Action
					users   []map[string]string
				)
				for _, action := range tc.actions {
					actions = append(actions, *action)
					for _, user := range action.Parameters.Users {
						users = append(users, user)
					}
				}

				// Create an actor store.
				actors := actor.NewStore(logger)
				// Set its actions and check its returned error according to the test
				// case.
				err := actors.SetActions(actions)
				require.NoError(t, err)

				// Test the store in parallel as it is meant to be thread-safe.
				t.Run("FindUser", func(t *testing.T) {
					// Accesses that should be successful.
					for _, user := range users {
						user := user // new scope for the following closure
						t.Run("FindUser", func(t *testing.T) {
							t.Parallel()
							t.Logf("FindUser(%#v)", user)
							got, exists := actors.FindUser(user)
							require.True(t, exists)
							require.Equal(t, got.ActionID(), findOriginalUserActionID(user, actions))
						})
					}

					// Accesses that should fail
					t.Run("Failed FindUser", func(t *testing.T) {
						t.Parallel()
						_, exists := actors.FindUser(RandUser())
						require.False(t, exists)
					})
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
				*NewTimedBlockUserAction(1, map[string]string{"uid": "oh my uid"}), // Enough time to perform the test
				*NewBlockUserAction(map[string]string{"uid": "oh my uid 2"}),
				*NewTimedBlockUserAction(1, map[string]string{"uid": "oh my uid 3"}), // Enough time to perform the test
			}
			err := actors.SetActions(actions)
			require.NoError(t, err)

			// Accesses that should be successful.

			// Start with timed actions first so that they shouldn't have expired yet
			timed := actions[0]
			timedUserID1 := timed.Parameters.Users[0]
			got, exists := actors.FindUser(timedUserID1)
			require.True(t, exists)
			require.Equal(t, got.ActionID(), timed.ActionId)

			timed = actions[2]
			timedUserID2 := timed.Parameters.Users[0]
			got, exists = actors.FindUser(timedUserID2)
			require.True(t, exists)
			require.Equal(t, got.ActionID(), timed.ActionId)

			// Wait for the timed duration to make sure it has expired
			time.Sleep(time.Duration(timed.Duration) * time.Second)
			// Tests should be false now
			_, exists = actors.FindUser(timedUserID1)
			require.False(t, exists) // The user does not exist
			_, exists = actors.FindUser(timedUserID2)
			require.False(t, exists) // The user does not exist

			notTimed := actions[1]
			notTimedUserID := notTimed.Parameters.Users[0]
			got, exists = actors.FindUser(notTimedUserID)
			require.True(t, exists)
			require.Equal(t, got.ActionID(), notTimed.ActionId)
		})
	})

	t.Run("IP", func(t *testing.T) {
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
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(80, 64, 3, 221), RandIPv4(), RandIPv4(), RandIPv4(), RandIPv6(), RandIPv6(), RandIPv6()},
			},

			{
				name: "Number of bits too high for IPv4",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/35"): nil,
				},
			},

			{
				name: "Number of bits too high for IPv6",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1:2:3:4::33/450"): nil,
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
					NewBlockIPAction("[1:2:3:4::]:80"): nil, // The store API does not expect `ip:port`
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
					NewBlockIPAction("1:2.2:3:4/42"): nil,
				},
			},

			{
				name: "Malformed",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1:2:3:4:0jul:0"): nil,
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
					NewBlockIPAction(testlib.RandString(2, 20)): nil,
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
				name: "1:2:3::4/128",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1:2:3::4/128"): []net.IP{net.ParseIP("1:2:3::4")},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 4), net.ParseIP("1:2:3::5"), net.ParseIP("1:2:3::3")},
			},

			{
				name: "1.2.3.4/32",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4/32", "1.2.3.4"): []net.IP{net.IPv4(1, 2, 3, 4)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3)},
			},

			{
				name: "Overalpping IPv4 Networks",
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
				name: "Overalpping IPv6 Networks",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1::/16", "1:00ff::/32", "1:ff00::/24"): []net.IP{
						net.ParseIP("1::"),
						net.ParseIP("1::42"),
						net.ParseIP("1::ffff:ffff"),
						net.ParseIP("1:ff00::"),
						net.ParseIP("1:ff00::42"),
						net.ParseIP("1:ff00::ffff:ffff"),
						net.ParseIP("1:00ff::"),
						net.ParseIP("1:00ff::42"),
						net.ParseIP("1:00ff::ffff:ffff"),
					},
				},
				findIPFailure: []net.IP{
					net.IPv4(1, 1, 255, 255),
					net.IPv4(1, 3, 0, 0),
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

			{
				name: "Overlapping IPv4 Networks",
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
				name: "Overlapping IPv6 Networks",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1::/32"): []net.IP{net.ParseIP("1::")},
					NewBlockIPAction("1:00ff::/24"): []net.IP{
						net.ParseIP("1:00ff::"),
						net.ParseIP("1:00ff::42"),
						net.ParseIP("1:00ff::ffff"),
					},
					NewBlockIPAction("1:ff00::/16"): []net.IP{
						net.ParseIP("1:ff00::"),
						net.ParseIP("1:ff00::dead"),
						net.ParseIP("1:ff00::ffff"),
					},
				},
				findIPFailure: []net.IP{
					net.IPv4(1, 1, 255, 255), net.IPv4(1, 3, 0, 0),
					net.ParseIP("2::"),
					net.ParseIP("ffff::"),
					net.ParseIP("0::"),
				},
			},

			{
				name: "Subsequent IPv4 CIDRs in a single action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.4", "1.2.3.5", "1.2.3.3"): []net.IP{net.IPv4(1, 2, 3, 5), net.IPv4(1, 2, 3, 3), net.IPv4(1, 2, 3, 4)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6)},
			},

			{
				name: "Subsequent IPv6 CIDRs in a single action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1:2:3::4", "1:2:3::5", "1:2:3::3"): []net.IP{net.ParseIP("1:2:3::5"), net.ParseIP("1:2:3::3"), net.ParseIP("1:2:3::4")},
				},
				findIPFailure: []net.IP{
					net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6),
					net.ParseIP("1:2:3::2"), net.ParseIP("1:2:3::6"),
				},
			},

			{
				name: "Subsequent IPv4 CIDRs per action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1.2.3.5"): []net.IP{net.IPv4(1, 2, 3, 5)},
					NewBlockIPAction("1.2.3.4"): []net.IP{net.IPv4(1, 2, 3, 4)},
					NewBlockIPAction("1.2.3.3"): []net.IP{net.IPv4(1, 2, 3, 3)},
				},
				findIPFailure: []net.IP{net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6)},
			},

			{
				name: "Subsequent IPv6 CIDRs per action",
				actions: map[*api.ActionsPackResponse_Action][]net.IP{
					NewBlockIPAction("1:2:3::5"): []net.IP{net.ParseIP("1:2:3::5")},
					NewBlockIPAction("1:2:3::4"): []net.IP{net.ParseIP("1:2:3::4")},
					NewBlockIPAction("1:2:3::3"): []net.IP{net.ParseIP("1:2:3::3")},
				},
				findIPFailure: []net.IP{
					net.IPv4(1, 2, 3, 2), net.IPv4(1, 2, 3, 6),
					net.ParseIP("1:2:3::2"), net.ParseIP("1:2:3::6"),
				},
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
				*NewTimedBlockIPAction(1, "1.2.3.4/32"), // Enough time to perform the test
				*NewBlockIPAction("1.2.3.4/24"),
				*NewTimedBlockIPAction(1, "1.2.3.4/16"), // Enough time to perform the test
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
			time.Sleep(time.Duration(timed.Duration) * time.Second)
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

func RandUser() map[string]string {
	count := int(1 + rand.Uint32()%10)
	user := make(map[string]string, count)
	for n := 0; n < count; n++ {
		k := testlib.RandString(1, 50)
		v := testlib.RandString(1, 50)
		user[k] = v
	}
	return user
}

func findOriginalUserActionID(userID map[string]string, actions []api.ActionsPackResponse_Action) string {
	for _, action := range actions {
		for _, user := range action.Parameters.Users {
			eq := reflect.DeepEqual(userID, user)
			if eq {
				return action.ActionId
			}
		}
	}
	return ""
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

func NewTimedBlockIPAction(d float64, CIDRs ...string) *api.ActionsPackResponse_Action {
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

func NewTimedBlockUserAction(d time.Duration, users ...map[string]string) *api.ActionsPackResponse_Action {
	action := NewBlockUserAction(users...)
	action.Duration = float64(d)
	return action
}

func NewBlockUserAction(users ...map[string]string) *api.ActionsPackResponse_Action {
	userList := make([]map[string]string, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	action := &api.ActionsPackResponse_Action{
		Action: "block_user",
		Parameters: api.ActionsPackResponse_Action_Params{
			Users: userList,
		},
	}
	fuzzer.Fuzz(&action.ActionId)
	fuzzer.Fuzz(&action.SendResponse)
	return action
}

func BenchmarkUserStore(b *testing.B) {
	b.Run("Lookup", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n
			store, users := RandUserStore(b, n, RandUser)
			b.Run(fmt.Sprintf("%d", len(users)), func(b *testing.B) {
				b.ReportAllocs()
				for n := 0; n < b.N; n++ {
					// Pick a random user that was inserted
					ix := int(rand.Int63n(int64(len(users))))
					user := users[ix]
					_, exists := store.FindUser(user)
					if !exists {
						b.FailNow()
					}
				}
			})
		}
	})

	b.Run("Insertion", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n
			actions, _ := RandUserActions(b, n, RandUser)
			store := actor.NewStore(logger)
			b.Run(fmt.Sprint(n), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					store.SetActions(actions)
				}
			})
		}
	})

	b.Run("Size", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n
			actions, _ := RandUserActions(b, n, RandUser)
			b.Run(fmt.Sprint(n), func(b *testing.B) {
				b.ReportAllocs()
				for n := 0; n < b.N; n++ {
					store := actor.NewStore(logger)
					store.SetActions(actions)
				}
			})
		}
	})
}

func RandUserStore(b *testing.B, count int, randUser func() map[string]string) (store *actor.Store, users []map[string]string) {
	store = actor.NewStore(logger)
	actions, users := RandUserActions(b, count, RandUser)
	err := store.SetActions(actions)
	require.NoError(b, err)
	return store, users
}

func RandUserActions(b *testing.B, count int, randUser func() map[string]string) (actions []api.ActionsPackResponse_Action, users []map[string]string) {
	actions = make([]api.ActionsPackResponse_Action, 0, count)
	for i := 0; i < count; i++ {
		user := randUser()
		action := NewBlockUserAction(user)
		actions = append(actions, *action)
		users = append(users, user)
	}
	return
}

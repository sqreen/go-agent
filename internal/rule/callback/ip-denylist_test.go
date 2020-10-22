// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net"
	"testing"

	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/rule/callback/_testlib/mockups"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIPDenyListCallback(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		t.Run("Configuration errors", func(t *testing.T) {
			for _, tc := range []interface{}{
				nil,
				33,                             // wrong type
				([]interface{})(nil),           // empty list
				[]interface{}{},                // empty list
				[]interface{}{([]string)(nil)}, // empty list
				[]interface{}{[]string{}},      // empty list
			} {
				tc := tc
				t.Run("", func(t *testing.T) {
					r := &mockups.NativeRuleContextMockup{}
					defer r.AssertExpectations(t)

					cfg := &mockups.NativeCallbackConfigMockup{}
					cfg.ExpectData().Return(tc)
					defer cfg.AssertExpectations(t)

					_, err := callback.NewIPDenyListCallback(r, cfg)
					require.Error(t, err)
				})
			}
		})
	})

	t.Run("Callback", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}

		cfg := &mockups.NativeCallbackConfigMockup{}
		// Note that exhaustive tests of the underlying IP list is done in the
		// corresponding package, and that we are only testing the callback API here
		// with a few examples.
		data := []interface{}{
			[]string{
				"1.2.3.4",
				"10.0.0.0/8",
			},
		}
		cfg.ExpectData().Return(data).Once()
		defer cfg.AssertExpectations(t)

		cb, err := callback.NewIPDenyListCallback(r, cfg)
		require.NoError(t, err)
		require.NotNil(t, cb)

		prolog, ok := cb.(callback.IPDenyListPrologCallbackType)
		require.True(t, ok)

		// Not blocked
		r.On("Pre", mock.MatchedBy(func(cb func(c callback.CallbackContext)) bool {
			c := &mockups.CallbackContextMockup{}
			defer c.AssertExpectations(t)

			p := &mockups.ProtectionContextMockup{}
			defer p.AssertExpectations(t)

			c.ExpectProtectionContext().Return(p)
			p.ExpectClientIP().Return(net.ParseIP("11.22.33.44")).Once()

			cb(c)
			return true
		})).Once()

		epilog, err := prolog(nil)
		require.Nil(t, epilog)
		require.NoError(t, err)

		r.AssertExpectations(t)

		// Blocked
		//ipStr := "1.2.3.4"
		//ip := net.ParseIP(ipStr)
		//requestReader.ExpectClientIP().Return(ip).Once()
		//r.ExpectPushMetricsValue(ipStr, 1).Return(nil).Once()
		//epilog, err = prolog(nil)
		//require.NotNil(t, epilog)
		//require.NoError(t, err)
		//requestReader.AssertExpectations(t)
		//r.AssertExpectations(t)

		//var e error
		//epilog(&e)
		//require.NoError(t, err)
		//require.True(t, errors.As(e, &sdktypes.SqreenError{}))
		//var actualErr callback.IPDenyListError
		//require.True(t, errors.As(e, &actualErr))
		//require.Equal(t, ipStr, actualErr.DenyListEntry)
		//require.Equal(t, ip, actualErr.DeniedIP)
	})
}

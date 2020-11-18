// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"testing"

	"github.com/sqreen/go-agent/internal/event"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	http_protection_mockups "github.com/sqreen/go-agent/internal/protection/http/_testlib/mockups"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/rule/callback/_testlib/mockups"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMonitorHTTPStatusCodeCallback(t *testing.T) {
	t.Run("not 404", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		r.ExpectPre(mock.Anything).Once()

		cb, err := callback.NewMonitorHTTPStatusCodeCallback(r, nil)
		require.NoError(t, err)

		prolog := cb.(http_protection.ResponseMonitoringPrologCallbackType)
		require.NotNil(t, prolog)

		resp := &http_protection_mockups.ResponseMockup{}
		defer resp.AssertExpectations(t)
		expectedStatusCode := http.StatusExpectationFailed
		resp.ExpectStatus().Return(expectedStatusCode).Once()

		var respFace types.ResponseFace = resp
		epilog, err := prolog(nil, &respFace)
		require.NoError(t, err)
		require.Nil(t, epilog)

		r.AssertCalled(t, "Pre", mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			defer c.AssertExpectations(t)
			c.ExpectAddMetricsValue(expectedStatusCode, 1).Return(true).Once()
			require.NoError(t, cb(c))
			return true
		}))
	})

	t.Run("404", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		r.ExpectPre(mock.Anything).Once()

		cb, err := callback.NewMonitorHTTPStatusCodeCallback(r, nil)
		require.NoError(t, err)

		prolog := cb.(http_protection.ResponseMonitoringPrologCallbackType)
		require.NotNil(t, prolog)

		resp := &http_protection_mockups.ResponseMockup{}
		defer resp.AssertExpectations(t)
		expectedStatusCode := http.StatusNotFound
		resp.ExpectStatus().Return(expectedStatusCode).Once()

		var respFace types.ResponseFace = resp
		epilog, err := prolog(nil, &respFace)
		require.NoError(t, err)
		require.Nil(t, epilog)

		r.AssertCalled(t, "Pre", mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			defer c.AssertExpectations(t)
			c.ExpectAddMetricsValue(expectedStatusCode, 1).Return(true).Once()
			c.ExpectHandleAttack(false, mock.MatchedBy(func(opts []event.AttackEventOption) bool {
				attack := &event.AttackEvent{}
				for _, opt := range opts {
					opt(attack)
				}
				return assert.Equal(t, &event.AttackEvent{Test: true, Info: struct{}{}}, attack)
			})).Return(false).Once()
			require.NoError(t, cb(c))
			return true
		}))
	})
}

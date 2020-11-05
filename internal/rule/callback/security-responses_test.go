// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"errors"
	"net"
	"net/http"
	"testing"

	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	http_protection_mockups "github.com/sqreen/go-agent/internal/protection/http/_testlib/mockups"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/rule/callback/_testlib/mockups"
	middleware_mockups "github.com/sqreen/go-agent/sdk/middleware/_testlib/mockups"
	"github.com/sqreen/go-agent/sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

type (
	ActionMockup struct {
		mock.Mock
	}

	RedirectionActionMockup struct {
		ActionMockup
	}
)

func (a *ActionMockup) ActionID() string {
	return a.Called().String(0)
}

func (a *ActionMockup) ExpectActionID() *mock.Call {
	return a.On("ActionID")
}

func (a *RedirectionActionMockup) RedirectionURL() string {
	return a.Called().String(0)
}

func (a *RedirectionActionMockup) ExpectRedirectionURL() *mock.Call {
	return a.On("RedirectionURL")
}

func TestIPSecurityResponseCallback(t *testing.T) {
	t.Run("not blocked", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
		defer rootCtx.AssertExpectations(t)

		responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
		defer responseWriterMockup.AssertExpectations(t)

		requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
		defer requestReaderMockup.AssertExpectations(t)

		ip := net.ParseIP("1.2.3.4")
		p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

		v, err := callback.NewIPSecurityResponseCallback(r, nil /* unused */)
		require.NoError(t, err)

		rootCtx.ExpectFindActionByIP(ip).Return(nil, false, nil)

		r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			require.NoError(t, cb(c))
			return true
		}))

		prolog := v.(http_protection.BlockingPrologCallbackType)
		epilog, err := prolog(&p)
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

	t.Run("blocked", func(t *testing.T) {
		t.Run("with default blocking behaviour", func(t *testing.T) {
			r := &mockups.NativeRuleContextMockup{}
			defer r.AssertExpectations(t)

			rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
			defer rootCtx.AssertExpectations(t)

			responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			ip := net.ParseIP("1.2.3.4")
			p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

			v, err := callback.NewIPSecurityResponseCallback(r, nil /* unused */)
			require.NoError(t, err)

			actionMockup := &ActionMockup{}
			defer actionMockup.AssertExpectations(t)

			rootCtx.ExpectFindActionByIP(ip).Return(actionMockup, true, nil)
			rootCtx.ExpectCancelContext()

			r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
				c := &mockups.CallbackContextMockup{}
				require.NoError(t, cb(c))
				return true
			}))

			prolog := v.(http_protection.BlockingPrologCallbackType)
			epilog, err := prolog(&p)
			require.NoError(t, err)
			require.NotNil(t, epilog)

			epilog(&err)
			require.Error(t, err)
			require.True(t, xerrors.As(err, &types.SqreenError{}))
		})

		t.Run("with redirection action", func(t *testing.T) {
			r := &mockups.NativeRuleContextMockup{}
			defer r.AssertExpectations(t)

			rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
			defer rootCtx.AssertExpectations(t)

			responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			ip := net.ParseIP("1.2.3.4")
			p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

			v, err := callback.NewIPSecurityResponseCallback(r, nil /* unused */)
			require.NoError(t, err)

			actionMockup := &RedirectionActionMockup{}
			defer actionMockup.AssertExpectations(t)
			expectedLocation := "https://sqreen.com/"
			actionMockup.ExpectRedirectionURL().Return(expectedLocation)

			headers := http.Header{}
			responseWriterMockup.ExpectHeader().Return(headers)
			responseWriterMockup.ExpectWriteHeader(http.StatusSeeOther)

			rootCtx.ExpectFindActionByIP(ip).Return(actionMockup, true, nil)
			rootCtx.ExpectCancelContext()

			r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
				c := &mockups.CallbackContextMockup{}
				require.NoError(t, cb(c))
				return true
			}))

			prolog := v.(http_protection.BlockingPrologCallbackType)
			epilog, err := prolog(&p)
			require.NoError(t, err)
			require.NotNil(t, epilog)

			require.Equal(t, expectedLocation, headers.Get("Location"))

			epilog(&err)
			require.Error(t, err)
			require.True(t, xerrors.As(err, &types.SqreenError{}))
		})
	})

	t.Run("ip lookup error", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
		defer rootCtx.AssertExpectations(t)

		responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
		defer responseWriterMockup.AssertExpectations(t)

		requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
		defer requestReaderMockup.AssertExpectations(t)

		ip := net.ParseIP("1.2.3.4")
		p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

		v, err := callback.NewIPSecurityResponseCallback(r, nil /* unused */)
		require.NoError(t, err)

		rootCtx.ExpectFindActionByIP(ip).Return(nil, false, errors.New("an error"))

		r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			err := cb(c)
			require.Error(t, err)
			return true
		}))

		prolog := v.(http_protection.BlockingPrologCallbackType)
		epilog, err := prolog(&p)
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

}

func TestUserSecurityResponseCallback(t *testing.T) {
	t.Run("not blocked", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
		defer rootCtx.AssertExpectations(t)

		responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
		defer responseWriterMockup.AssertExpectations(t)

		requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
		defer requestReaderMockup.AssertExpectations(t)

		p := http_protection.NewTestProtectionContext(rootCtx, net.ParseIP("1.2.3.4"), responseWriterMockup, requestReaderMockup)

		v, err := callback.NewUserSecurityResponseCallback(r, nil /* unused */)
		require.NoError(t, err)

		userID := map[string]string{
			"uid": "unique user id",
		}
		rootCtx.ExpectFindActionByUserID(userID).Return(nil, false, nil)

		r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			require.NoError(t, cb(c))
			return true
		}))

		prolog := v.(http_protection.IdentifyUserPrologCallbackType)
		epilog, err := prolog(&p, &userID)
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

	t.Run("blocked", func(t *testing.T) {
		t.Run("with default blocking behaviour", func(t *testing.T) {
			r := &mockups.NativeRuleContextMockup{}
			defer r.AssertExpectations(t)

			rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
			defer rootCtx.AssertExpectations(t)

			responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			ip := net.ParseIP("1.2.3.4")
			p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

			v, err := callback.NewUserSecurityResponseCallback(r, nil /* unused */)
			require.NoError(t, err)

			actionMockup := &ActionMockup{}
			defer actionMockup.AssertExpectations(t)

			userID := map[string]string{
				"uid": "unique user id",
			}
			rootCtx.ExpectFindActionByUserID(userID).Return(actionMockup, true)
			rootCtx.ExpectCancelContext()

			r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
				c := &mockups.CallbackContextMockup{}
				require.NoError(t, cb(c))
				return true
			}))

			prolog := v.(http_protection.IdentifyUserPrologCallbackType)
			epilog, err := prolog(&p, &userID)
			require.NoError(t, err)
			require.NotNil(t, epilog)

			epilog(&err)
			require.Error(t, err)
			require.True(t, xerrors.As(err, &types.SqreenError{}))
		})

		t.Run("with redirection action", func(t *testing.T) {
			r := &mockups.NativeRuleContextMockup{}
			defer r.AssertExpectations(t)

			rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
			defer rootCtx.AssertExpectations(t)

			responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
			defer responseWriterMockup.AssertExpectations(t)

			requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
			defer requestReaderMockup.AssertExpectations(t)

			ip := net.ParseIP("1.2.3.4")
			p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

			v, err := callback.NewUserSecurityResponseCallback(r, nil /* unused */)
			require.NoError(t, err)

			actionMockup := &RedirectionActionMockup{}
			defer actionMockup.AssertExpectations(t)
			expectedLocation := "https://sqreen.com/"
			actionMockup.ExpectRedirectionURL().Return(expectedLocation)

			headers := http.Header{}
			responseWriterMockup.ExpectHeader().Return(headers)
			responseWriterMockup.ExpectWriteHeader(http.StatusSeeOther)

			userID := map[string]string{
				"uid": "unique user id",
			}
			rootCtx.ExpectFindActionByUserID(userID).Return(actionMockup, true)
			rootCtx.ExpectCancelContext()

			r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
				c := &mockups.CallbackContextMockup{}
				require.NoError(t, cb(c))
				return true
			}))

			prolog := v.(http_protection.IdentifyUserPrologCallbackType)
			epilog, err := prolog(&p, &userID)
			require.NoError(t, err)
			require.NotNil(t, epilog)

			require.Equal(t, expectedLocation, headers.Get("Location"))

			epilog(&err)
			require.Error(t, err)
			require.True(t, xerrors.As(err, &types.SqreenError{}))
		})
	})

	t.Run("ip lookup error", func(t *testing.T) {
		r := &mockups.NativeRuleContextMockup{}
		defer r.AssertExpectations(t)

		rootCtx := &middleware_mockups.RootHTTPProtectionContextMockup{}
		defer rootCtx.AssertExpectations(t)

		responseWriterMockup := &http_protection_mockups.ResponseWriterMockup{}
		defer responseWriterMockup.AssertExpectations(t)

		requestReaderMockup := &http_protection_mockups.RequestReaderMockup{}
		defer requestReaderMockup.AssertExpectations(t)

		ip := net.ParseIP("1.2.3.4")
		p := http_protection.NewTestProtectionContext(rootCtx, ip, responseWriterMockup, requestReaderMockup)

		v, err := callback.NewIPSecurityResponseCallback(r, nil /* unused */)
		require.NoError(t, err)

		rootCtx.ExpectFindActionByIP(ip).Return(nil, false, errors.New("an error"))

		r.ExpectPre(mock.MatchedBy(func(cb func(callback.CallbackContext) error) bool {
			c := &mockups.CallbackContextMockup{}
			err := cb(c)
			require.Error(t, err)
			return true
		}))

		prolog := v.(http_protection.BlockingPrologCallbackType)
		epilog, err := prolog(&p)
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

}

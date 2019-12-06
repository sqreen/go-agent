// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestAction(t *testing.T) {
	t.Run("Blocking action", func(t *testing.T) {
		action := newBlockAction(testlib.RandPrintableUSASCIIString(1, 20))

		t.Run("with duration", func(t *testing.T) {
			t.Run("not expired", func(t *testing.T) {
				action := withDuration(action, 10*time.Hour)
				require.False(t, action.Expired())
			})
			t.Run("expired", func(t *testing.T) {
				action := withDuration(action, 0)
				require.True(t, action.Expired())
			})
		})

		t.Run("HTTP Handler", func(t *testing.T) {
			t.Run("Block IP", func(t *testing.T) {
				handler, err := NewIPActionHTTPHandler(action, net.IPv4(1, 2, 3, 4))
				require.NotNil(t, handler)
				require.Nil(t, err)
				// Use the handler
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				require.Equal(t, rec.Code, 500)
				// TODO: check the sdk event
			})

			t.Run("Block User", func(t *testing.T) {
				handler, err := NewUserActionHTTPHandler(action, map[string]string{"uid": testlib.RandPrintableUSASCIIString(1, 250)})
				require.NoError(t, err)
				require.NotNil(t, handler)
				// Use the handler
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				require.Equal(t, rec.Code, 500)
				// TODO: check the sdk event
			})
		})
	})

	t.Run("Redirection action", func(t *testing.T) {
		t.Run("invalid location url", func(t *testing.T) {
			action, err := newRedirectAction(testlib.RandPrintableUSASCIIString(1, 20), "http//toto")
			require.Nil(t, action)
			require.Error(t, err)
		})

		t.Run("valid location url", func(t *testing.T) {
			action, err := newRedirectAction(testlib.RandPrintableUSASCIIString(1, 20), "http://sqreen.com")
			require.NotNil(t, action)
			require.NoError(t, err)

			t.Run("with duration", func(t *testing.T) {
				t.Run("not expired", func(t *testing.T) {
					action := withDuration(action, 10*time.Hour)
					require.False(t, action.Expired())
				})
				t.Run("expired", func(t *testing.T) {
					action := withDuration(action, 0)
					require.True(t, action.Expired())
				})
			})

			t.Run("HTTP Handler", func(t *testing.T) {
				t.Run("Redirect IP", func(t *testing.T) {
					handler, err := NewIPActionHTTPHandler(action, net.IPv4(1, 2, 3, 4))
					require.NotNil(t, handler)
					require.Nil(t, err)
					// Use the handler
					req := httptest.NewRequest(http.MethodPost, "/", nil)
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, req)
					require.Equal(t, rec.Code, http.StatusSeeOther)
					require.Equal(t, rec.Header().Get("Location"), action.URL)
					// TODO: check the sdk event
				})

				t.Run("Redirect User", func(t *testing.T) {
					handler, err := NewUserActionHTTPHandler(action, map[string]string{"uid": testlib.RandPrintableUSASCIIString(1, 250)})
					require.NoError(t, err)
					require.NotNil(t, handler)
					// Use the handler
					req := httptest.NewRequest(http.MethodPost, "/", nil)
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, req)
					require.Equal(t, rec.Code, http.StatusSeeOther)
					require.Equal(t, rec.Header().Get("Location"), action.URL)
					// TODO: check the sdk event
				})
			})
		})
	})
}

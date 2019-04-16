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
	action := newBlockAction(testlib.RandString(1, 20))

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
			handler := NewIPActionHTTPHandler(action, net.IPv4(1, 2, 3, 4))
			require.NotNil(t, handler)
			// Use the handler
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, rec.Code, 500)
			// TODO: check the sdk event
		})

		t.Run("Block User", func(t *testing.T) {
			handler := NewUserActionHTTPHandler(action, map[string]string{"uid": testlib.RandString(1, 250)})
			require.NotNil(t, handler)
			// Use the handler
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, rec.Code, 500)
			// TODO: check the sdk event
		})
	})
}

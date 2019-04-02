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

func TestActionHandler(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		handler := NewActionHandler(nil, nil)
		require.Nil(t, handler)
	})

	t.Run("BlockIPAction", func(t *testing.T) {
		action := newBlockIPAction(testlib.RandString(1, 20))
		handler := NewActionHandler(action, net.IPv4(1, 2, 3, 4))
		require.NotNil(t, handler)

		t.Run("is a http.Handler", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, rec.Code, 500)
		})

		t.Run("is a types.EventProperties", func(t *testing.T) {
			props := (*blockedIPEventProperties)(handler.(*blockIPActionHandler))
			_, err := props.MarshalJSON()
			require.NoError(t, err)
		})

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
	})
}

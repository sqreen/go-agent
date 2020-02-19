// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	Time "time"

	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Run("using the sdk without middleware", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandUTF8String(4096)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			sq := sdk.FromContext(req.Context())
			require.NotNil(t, sq)
			sq.TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(Time.Now()).WithUserIdentifiers(sdk.EventUserIdentifiersMap{"my": "id"})
			sq.ForUser(sdk.EventUserIdentifiersMap{"my": "id"}).TrackEvent("my event").WithProperties(sdk.EventPropertyMap{"my": "prop"}).WithTimestamp(Time.Now())
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})
		router.Handle("/", subrouter)
		// Perform the requestImplType and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the requestImplType was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})
}

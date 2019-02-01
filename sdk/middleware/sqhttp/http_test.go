package sqhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	require := require.New(t)
	body := testlib.RandString(1, 100)
	// Create a router
	router := http.NewServeMux()
	// Add an endpoint accessing the SDK handle
	subrouter := http.NewServeMux()
	subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
		require.NotNil(sdk.FromContext(req.Context()), "The middleware should attach its handle object to the request's context")
		w.Write([]byte(body))
		w.WriteHeader(http.StatusOK)
	})
	router.Handle("/", sqhttp.Middleware(subrouter))
	// Perform the request and record the output
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)
	router.ServeHTTP(rec, req)
	// Check the request was performed as expected
	require.Equal(http.StatusOK, rec.Code)
	require.Equal(body, rec.Body.String())
}

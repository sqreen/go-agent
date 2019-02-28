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
	req, _ := http.NewRequest("GET", "/hello", nil)

	agent := testlib.NewAgentForMiddlewareTests(req)
	sdk.SetAgent(agent)

	t.Run("without security action", func(t *testing.T) {
		defer agent.AssertExpectations(t)

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
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(http.StatusOK, rec.Code)
		require.Equal(body, rec.Body.String())
	})

	t.Run("with security action", func(t *testing.T) {
		agent.ResetExpectations()
		defer agent.AssertExpectations(t)

		status := http.StatusBadRequest
		action := testlib.NewSecurityActionBlockWithStatus(status)
		defer action.AssertExpectations(t)

		agent.ExpectSecurityAction(req).Return(action).Once()

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
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(rec.Code, status)
		require.Equal(rec.Body.String(), "")
	})
}

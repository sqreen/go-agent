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
	t.Run("without agent", func(t *testing.T) {
		sdk.SetAgent(nil)

		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandString(1, 100)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			require.Nil(t, sdk.FromContext(req.Context()))
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})
		router.Handle("/", sqhttp.Middleware(subrouter))
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without middleware", func(t *testing.T) {
		agent := &testlib.AgentMockup{}
		defer agent.AssertExpectations(t)
		sdk.SetAgent(agent)

		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandString(1, 100)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			require.Nil(t, sdk.FromContext(req.Context()))
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})
		router.Handle("/", subrouter)
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("without security action", func(t *testing.T) {
		agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityAction()
		sdk.SetAgent(agent)
		defer agent.AssertExpectations(t)
		defer record.AssertExpectations(t)

		req, _ := http.NewRequest("GET", "/hello", nil)
		body := testlib.RandString(1, 100)
		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			require.NotNil(t, sdk.FromContext(req.Context()), "The middleware should attach its handle object to the request's context")
			w.Write([]byte(body))
			w.WriteHeader(http.StatusOK)
		})
		router.Handle("/", sqhttp.Middleware(subrouter))
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, body, rec.Body.String())
	})

	t.Run("with security action", func(t *testing.T) {
		status := http.StatusBadRequest
		agent, record := testlib.NewAgentForMiddlewareTestsWithSecurityAction(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		sdk.SetAgent(agent)
		defer agent.AssertExpectations(t)
		defer record.AssertExpectations(t)

		// Create a router
		router := http.NewServeMux()
		// Add an endpoint accessing the SDK handle
		subrouter := http.NewServeMux()
		subrouter.HandleFunc("/hello", func(w http.ResponseWriter, req *http.Request) {
			panic("must not be called")
		})
		router.Handle("/", sqhttp.Middleware(subrouter))
		// Perform the request and record the output
		req, _ := http.NewRequest("GET", "/hello", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(t, rec.Body.String(), "")
		require.Equal(t, rec.Code, status)
	})
}

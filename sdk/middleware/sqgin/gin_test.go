package sqgin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqgin"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)

	agent := testlib.NewAgentForMiddlewareTests(req)
	sdk.SetAgent(agent)

	t.Run("without security action", func(t *testing.T) {
		defer agent.AssertExpectations(t)

		require := require.New(t)
		body := testlib.RandString(1, 100)
		// Create a Gin router
		router := gin.New()
		// Attach our middelware
		router.Use(sqgin.Middleware())
		// Add an endpoint accessing the SDK handle
		router.GET("/", func(c *gin.Context) {
			require.NotNil(sdk.FromContext(c), "The middleware should attach its handle object to Gin's context")
			require.NotNil(sdk.FromContext(c.Request.Context()), "The middleware should attach its handle object to the request's context")
			c.String(http.StatusOK, body)
		})
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
		// Create a Gin router
		router := gin.New()
		// Attach our middelware
		router.Use(sqgin.Middleware())
		// Add an endpoint accessing the SDK handle
		router.GET("/", func(c *gin.Context) {
			require.NotNil(sdk.FromContext(c), "The middleware should attach its handle object to Gin's context")
			require.NotNil(sdk.FromContext(c.Request.Context()), "The middleware should attach its handle object to the request's context")
			c.String(http.StatusOK, body)
		})
		// Perform the request and record the output
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		// Check the request was performed as expected
		require.Equal(rec.Code, status)
		require.Equal(rec.Body.String(), "")
	})
}

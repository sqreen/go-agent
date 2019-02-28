package sqecho_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqecho"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	body := testlib.RandString(1, 100)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	agent := testlib.NewAgentForMiddlewareTests(req)
	sdk.SetAgent(agent)

	t.Run("without security action", func(t *testing.T) {
		agent.ResetExpectations()
		defer agent.AssertExpectations(t)

		require := require.New(t)
		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			require.NotNil(sqecho.FromContext(c), "The middleware should attach its handle object to Gin's context")
			require.NotNil(sdk.FromContext(c.Request().Context()), "The middleware should attach its handle object to the request's context")
			body, err := ioutil.ReadAll(c.Request().Body)
			if err != nil {
				return err
			}
			return c.String(http.StatusOK, string(body))
		}
		// Perform the request and record the output
		mw := sqecho.Middleware()
		err := mw(h)(c)
		// Check the request was performed as expected
		require.NoError(err)
		require.Equal(http.StatusOK, rec.Code)
		require.Equal(body, rec.Body.String())
	})

	t.Run("with a security action", func(t *testing.T) {
		agent.ResetExpectations()
		defer agent.AssertExpectations(t)

		status := http.StatusBadRequest
		action := testlib.NewSecurityActionBlockWithStatus(status)
		defer action.AssertExpectations(t)

		agent.ExpectSecurityAction(req).Return(action).Once()

		require := require.New(t)

		// Create an Echo context
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// Define a Echo handler
		h := func(c echo.Context) error {
			panic("called")
			return c.String(http.StatusOK, testlib.RandString(10))
		}
		// Perform the request and record the output
		mw := sqecho.Middleware()
		err := mw(h)(c)
		// Check the request was performed as expected
		require.NoError(err)
		require.Equal(rec.Code, status)
		require.Equal(rec.Body.String(), "")
	})
}

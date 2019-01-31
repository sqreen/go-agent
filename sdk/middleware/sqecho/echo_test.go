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
	require := require.New(t)
	// Create an Echo context
	e := echo.New()
	hw := testlib.RandString(1, 100)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(hw))
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
	require.Equal(hw, rec.Body.String())
}

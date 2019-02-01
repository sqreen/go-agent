package sqiris_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqiris"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	require := require.New(t)
	hw := testlib.RandString(1, 100)
	// Create an iris app
	app := iris.New()
	// Attach our middelware
	app.Use(sqiris.Middleware())
	// Add an endpoint accessing the SDK handle
	app.Get("/", func(c iris.Context) {
		require.NotNil(sqiris.FromContext(c), "The middleware should attach its handle object to Iris' context")
		require.Nil(sdk.FromContext(c.Request().Context()))
		c.WriteString(hw)
	})

	// Perform the request, record the output and check the request was performed
	// as expected
	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal(hw)
}

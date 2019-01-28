package sqgin

import (
	"context"

	gingonic "github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for Gin to monitor and protect the
// requests Gin receives. It creates and stores the sdk's context into Gin's so
// that they can be retrieved from handlers to perform sdk calls using
// `GetHTTPContext()`.
//
//	router := gin.Default()
//	router.Use(sqgin.Middleware())
//
//	router.GET("/", func(c *gin.Context) {
//		sqgin.GetHTTPContext(c).TrackEvent("my.event.one")
//		aFunction(c.Request.Context())
//	}
//
//	func aFunction(ctx context.Context) {
//		sqgin.GetHTTPContext(ctx).TrackEvent("my.event.two")
//		// ...
//	}
func Middleware() gingonic.HandlerFunc {
	return func(c *gingonic.Context) {
		// Create a sqreen context for this request.
		sqreen := sdk.NewHTTPRequestContext(request{c.Copy()})

		// Store it into Go's context.
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, sdk.HTTPRequestContextKey, sqreen)
		c.Request = c.Request.WithContext(ctx)

		// Store it into Gin's context.
		c.Set(sdk.HTTPRequestContextKey, sqreen)

		c.Next()

		// Close the sqreen context
		sqreen.Close()
	}
}

type request struct {
	*gingonic.Context
}

func (r request) StdRequest() *http.Request {
	return r.Context.Request
}

package sqgin

import (
	"context"

	gingonic "github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for Gin to monitor and protect the
// requests Gin receives. It creates and stores the HTTP request record both
// into the Gin and request contexts so that it can be later accessed from
// handlers using `sdk.FromContext()` to perform SDK calls.
//
// Note that Gin's context implements the `context.Context` interface, so
// `sdk.FromContext()` can be used both with the Gin and request contexts.
//
//	router := gin.Default()
//	router.Use(sqgin.Middleware())
//
//	router.GET("/", func(c *gin.Context) {
//		// Accessing the SDK through Gin's context
//		sdk.FromContext(c).TrackEvent("my.event.one")
//		foo(c.Request)
//	}
//
//	func foo(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//		// ...
//	}
//
func Middleware() gingonic.HandlerFunc {
	return func(c *gingonic.Context) {
		// Create a new request record for this request.
		sqreen := sdk.NewHTTPRequestRecord(c.Request)
		defer sqreen.Close()

		// Gin redefines the request context interface, so we need to store it both
		// in the request and Gin contexts.

		// Store it into the request's context.
		contextKey := sdk.HTTPRequestRecordContextKey.String
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, contextKey, sqreen)
		c.Request = c.Request.WithContext(ctx)

		// Store it into Gin's context.
		c.Set(contextKey, sqreen)

		c.Next()
	}
}

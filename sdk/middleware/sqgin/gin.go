package sqgin

import (
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
		// Get current request.
		r := c.Request
		// Create a new sqreen request wrapper.
		req := sdk.NewHTTPRequest(r)
		defer req.Close()
		// Use the newly created request compliant with `sdk.FromContext()`.
		r = req.Request()
		// Also replace Gin's request pointer with it.
		c.Request = r

		// Check if a security action is required
		if handler := req.SecurityAction(); handler != nil {
			handler.ServeHTTP(c.Writer, r)
			c.Abort()
			return
		}

		// Gin implements the `context.Context` interface but with string keys, so
		// we need to also store the request record in Gin's context using a string
		// key (previous call to `sdk.NewHTTPRequest()` stored it with a non-string
		// key, as documented by `context.WithValue()`
		// (https://godoc.org/context#WithValue)).
		contextKey := sdk.HTTPRequestRecordContextKey.String
		c.Set(contextKey, req.Record())

		// Call next handler.
		c.Next()
	}
}

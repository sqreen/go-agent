package sqgin

import (
	gingonic "github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for Gin to monitor and protect the
// requests Gin receives. In protection mode, it can block and redirect requests
// according to its IP address or identified user using `Identify()` and
// `MatchSecurityResponse()` methods from the request handler.
//
// SDK methods can be called from request handlers by using the request event
// record. It can be accessed using `sdk.FromContext()` on a request context or
// on a Gin request context. The middleware function stores it into both of
// them. Note that Gin's context implements the `context.Context` interface
// which allows `sdk.FromContext()` to be used with both of them.
//
// Usage example:
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
//	router.GET("/", func(c *gin.Context) {
//		// Globally identifying a user and checking if the request should be
//		// aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sdk.FromContext(c).ForUser(uid)
//		sqUser.Identify() // Globally associate this user to the current request
//		if match, _ := sqUser.MatchSecurityResponse(); match {
//			// Return to stop further handling the request and let Sqreen's
//			// middleware apply and abort the request.
//			return
//		}
//		// ... not blocked ...
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

		// Check if an early security action is already required such as based on
		// the request IP address.
		if handler := req.SecurityResponse(); handler != nil {
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

		// Check if a security response should be applied now after having used
		// `Identify()` and `MatchSecurityResponse()`.
		if handler := req.UserSecurityResponse(); handler != nil {
			handler.ServeHTTP(c.Writer, r)
			c.Abort()
		}
	}
}

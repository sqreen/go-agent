// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin

import (
	"net/http"

	gingonic "github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
)

// Middleware is Sqreen's middleware function for Gin to monitor and protect the
// requests Gin receives. In protection mode, it can block and redirect requests
// according to their IP addresses or identified users using `Identify()` and
// `MatchSecurityResponse()` methods.
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
//		// Example of globally identifying a user and checking if the request
//		// should be aborted.
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
		// Adapt sqhttp middleware to Gin's
		err := sqhttp.MiddlewareWithError(sqhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			c.Request = r
			// Gin implements the `context.Context` interface but with string keys, so
			// we need to also store the request record in Gin's context using a string
			// key (previous call to `sdk.NewHTTPRequest()` stored it with a non-string
			// key, as documented by `context.WithValue()`
			// (https://godoc.org/context#WithValue)).
			contextKey := sdk.HTTPRequestRecordContextKey.String
			c.Set(contextKey, sdk.FromContext(r.Context()))
			c.Next()
			return nil
		})).ServeHTTP(c.Writer, c.Request)
		if err != nil {
			c.Abort()
		}
	}
}

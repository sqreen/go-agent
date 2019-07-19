// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
)

// Middleware is Sqreen's middleware function for Echo to monitor and protect
// the requests Echo receives. In protection mode, it can block and redirect
// requests according to their IP addresses or identified users using
// Identify()` and `MatchSecurityResponse()` methods.
//
// SDK methods can be called from request handlers by using the request event
// record. It can be accessed using `sdk.FromContext()` on a request context or
// this package's `FromContext()` on an Echo request context. The middleware
// function stores it into both of them.
//
// Usage example:
//
//	e := echo.New()
//	e.Use(sqecho.Middleware())
//
//	e.GET("/", func(c echo.Context) error {
//		// Accessing the SDK through Echo's context
//		sqecho.FromContext(c).TrackEvent("my.event.one")
//		foo(c.Request())
//		return nil
//	}
//
//	func foo(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//	}
//
//	e.GET("/", func(c echo.Context) {
//		// Globally identifying a user and checking if the request should be
//		// aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sqecho.FromContext(c).ForUser(uid)
//		sqUser.Identify() // Globally associate this user to the current request
//		if match, err := sqUser.MatchSecurityResponse(); match {
//			// Return to stop further handling the request and let Sqreen's
//			// middleware apply and abort the request.
//			return err
//		}
//		// ... not blocked ...
//		return nil
//	}
//
func Middleware() echo.MiddlewareFunc {
	// Create a middleware function by adapting to sqhttp's
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return sqhttp.MiddlewareWithError(sqhttp.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) error {
				c.SetRequest(r)
				// Echo defines its own context interface, so we need to store it in
				// Echo's context. Echo expects string keys.
				contextKey := sdk.HTTPRequestRecordContextKey.String
				c.Set(contextKey, sdk.FromContext(r.Context()))
				c.Response().After(func() {
					// Hack for now to monitor the status code because Echo doesn't use the
					// HTTP ResponseWriter when overwriting it through c.Writer = ...
					sqhttp.ResponseWriter{}.WriteHeader(c.Response().Status)
				})
				return next(c)
			})).ServeHTTP(c.Response(), c.Request())
		}
	}
}

// FromContext allows to access the HTTPRequestRecord from Echo request handlers
// if present, and nil otherwise. The value is stored in handler contexts by the
// middleware function, and is of type *HTTPRequestRecord.
//
// Note that Echo's context does not implement the `context.Context` interface,
// so `sdk.FromContext()` cannot be used with it, hence `FromContext()` in this
// package to access the SDK context value from Echo's context.
// `sdk.FromContext()` can still be used on the request context.
func FromContext(c echo.Context) *sdk.HTTPRequestRecord {
	v := c.Get(sdk.HTTPRequestRecordContextKey.String)
	if v == nil {
		return nil
	}
	return v.(*sdk.HTTPRequestRecord)
}

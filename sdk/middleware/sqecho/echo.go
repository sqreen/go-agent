// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
	"golang.org/x/xerrors"
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
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get current request.
			r := c.Request()
			// Create a new sqreen request wrapper.
			req := sdk.NewHTTPRequest(r)
			defer req.Close()
			// Use the newly created request compliant with `sdk.FromContext()`.
			r = req.Request()
			// Also replace Echo's request pointer with it.
			c.SetRequest(r)

			// Check if an early security action is already required such as based on
			// the request IP address.
			if handler := req.SecurityResponse(); handler != nil {
				handler.ServeHTTP(c.Response(), req.Request())
				return nil
			}

			// Echo defines its own context interface, so we need to store it in
			// Echo's context. Echo expects string keys.
			contextKey := sdk.HTTPRequestRecordContextKey.String
			c.Set(contextKey, req.Record())

			// Call next handler.
			err := next(c)
			if err != nil && !xerrors.As(err, &sdk.SecurityResponseMatch{}) {
				// The error is not a security response match
				return err
			}

			// Check if a security response should be applied now after having used
			// `Identify()` and `MatchSecurityResponse()`.
			if handler := req.UserSecurityResponse(); handler != nil {
				handler.ServeHTTP(c.Response(), req.Request())
			}

			return nil
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

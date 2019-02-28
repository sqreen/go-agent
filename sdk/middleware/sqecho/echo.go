package sqecho

import (
	"context"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for Echo to monitor and protect
// the requests Echo receives. It creates and stores the HTTP request record
// both into the Echo and request contexts so that it can be later accessed from
// handlers using `sdk.FromContext()` to perform SDK calls.
//
// Note that Echo's context does not implement the `context.Context` interface,
// so `sdk.FromContext()` cannot be used with it, hence `FromContext()` in this
// package to access the SDK context value from Echo's context.
// `sdk.FromContext()` can still be used on the request context.
//
//	e := echo.New()
//	e.Use(sqecho.Middleware())
//
//	e.GET("/", func(c echo.Context) {
//		// Accessing the SDK through Echo's context
//		sqecho.FromContext(c).TrackEvent("my.event.one")
//		foo(c.Request())
//	}
//
//	func foo(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//	}
//
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			if action := sdk.SecurityAction(req); action != nil {
				action.Apply(c.Response())
				return nil
			}

			// Create a new request record for this request.
			sqreen := sdk.NewHTTPRequestRecord(req)
			defer sqreen.Close()

			// Echo defines its own context interface, so we need to store it both in
			// the request and Echo contexts.

			// Store it into the request's context.
			contextKey := sdk.HTTPRequestRecordContextKey.String
			ctx := req.Context()
			ctx = context.WithValue(ctx, contextKey, sqreen)
			c.SetRequest(req.WithContext(ctx))

			// Store it into Echo's context.
			c.Set(contextKey, sqreen)

			return next(c)
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

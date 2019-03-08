package sqecho

import (
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
			// Get current request.
			r := c.Request()
			// Create a new sqreen request wrapper.
			req := sdk.NewHTTPRequest(r)
			defer req.Close()
			// Use the newly created request compliant with `sdk.FromContext()`.
			r = req.Request()
			// Also replace Echo's request pointer with it.
			c.SetRequest(r)

			// Check if a security action is required
			if handler := req.SecurityAction(); handler != nil {
				handler.ServeHTTP(c.Response(), req.Request())
				return nil
			}

			// Echo defines its own context interface, so we need to store it in
			// Echo's context. Echo expects string keys.
			contextKey := sdk.HTTPRequestRecordContextKey.String
			c.Set(contextKey, req.Record())

			// Call next handler.
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

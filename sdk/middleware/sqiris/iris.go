package sqiris

import (
	"github.com/kataras/iris"
	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for Iris to monitor and protect
// the requests Iris receives. It creates and stores the HTTP request record
// both into Iris' context so that it can be later accessed from handlers using
// `FromContext()` to perform SDK calls.
//
// Note that `sdk.FromContext()` cannot be used with Iris as the Iris context
// does not implement the `context.Context` inteface and also does not allow
// storing values into the request context. `FromContext()` defined in this
// package should be only used instead.
//
//	app := iris.New()
//	app.Use(sqiris.Middleware())
//
//	app.Get("/", func(c iris.Context) {
//		// Accessing the SDK through Iris context
//		sqiris.FromContext(c).TrackEvent("my.event.one")
//		// ...
//	}
//
func Middleware() iris.Handler {
	return func(c iris.Context) {
		// Create a new request record for this request.
		req := c.Request()
		sqreen := sdk.NewHTTPRequestRecord(req)
		defer sqreen.Close()

		// Store it into the request's context.
		contextKey := sdk.HTTPRequestRecordContextKey.String
		// Store it into Iris' context.
		c.Values().Set(contextKey, sqreen)

		c.Next()
	}
}

// FromContext allows to access the HTTPRequestRecord from Iris request handlers
// if present, and nil otherwise. The value is stored in handler contexts by the
// middleware function, and is of type *HTTPRequestRecord.
//
// This is the only way with Iris to access the HTTPRequestRecord and
// `sdk.FromContext()` cannot be used, as the Iris context does not implement
// the `context.Context` interface and also does not allow storing values into
// the request context.
func FromContext(c iris.Context) *sdk.HTTPRequestRecord {
	v := c.Values().Get(sdk.HTTPRequestRecordContextKey.String)
	if v == nil {
		return nil
	}
	return v.(*sdk.HTTPRequestRecord)
}

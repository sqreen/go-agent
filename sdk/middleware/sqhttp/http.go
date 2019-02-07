package sqhttp

import (
	"context"
	"net/http"

	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for `net/http` to monitor and
// protect received requests. It creates and stores the HTTP request record into
// the request context so that it can be later accessed to perform SDK calls in
// the decorated handler using `sdk.FromContext()`.
//
//	fn := func(w http.ResponseWriter, r *http.Request) {
//		sdk.FromContext(r.Context()).TrackEvent("my.event")
//		fmt.Fprintf(w, "OK")
//	}
//	http.Handle("/foo", sqhttp.Middleware(http.HandlerFunc(fn)))
//
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new request record for this request.
		sqreen := sdk.NewHTTPRequestRecord(r)
		defer sqreen.Close()

		// Store it into the request's context.
		ctx := r.Context()
		contextKey := sdk.HTTPRequestRecordContextKey.String
		ctx = context.WithValue(ctx, contextKey, sqreen)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

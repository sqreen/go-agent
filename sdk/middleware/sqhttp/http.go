package sqhttp

import (
	"net/http"

	"github.com/sqreen/go-agent/sdk"
)

// Middleware is Sqreen's middleware function for `net/http` to monitor and
// protect received requests. In protection mode, it can block and redirect
// requests according to its IP address or identified user using `Identify()`
// and `SecurityResponse()` methods during from the request handler.
//
// SDK methods can be called from request handlers by using the request event
// record. It can be accessed using `sdk.FromContext()` on a request context.
// The middleware function stores it into the request context.
//
// Usage example:
//
//	fn := func(w http.ResponseWriter, r *http.Request) {
//		// Get the request record.
//		sqreen := sdk.FromContext(r.Context())
//
//		// Example of sending a custom event.
//		sqreen.TrackEvent("my.event")
//
//		// Example of globally identifying a user and checking if the request
//		// should be aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sqreen.ForUser(uid)
//		sqUser.Identify() // Globally associate this user to the current request
//		if sqUser.MatchSecurityResponse() {
//			// Return to stop further handling the request and let Sqreen's
//			// middleware apply and abort the request.
//			return
//		}
//		// Not blocked.
//		fmt.Fprintf(w, "OK")
//	}
//	http.Handle("/foo", sqhttp.Middleware(http.HandlerFunc(fn)))
//
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new sqreen request wrapper.
		req := sdk.NewHTTPRequest(r)
		defer req.Close()
		// Use the newly created request compliant with `sdk.FromContext()`.
		r = req.Request()

		// Check if an early security action is already required such as based on
		// the request IP address.
		if handler := req.SecurityResponse(); handler != nil {
			handler.ServeHTTP(w, r)
			return
		}

		// Call next handler.
		next.ServeHTTP(w, r)

		// Check if a security response should be applied now after having used
		// `Identify()` and `MatchSecurityResponse()`.
		if handler := req.SecurityResponse(); handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
	})
}

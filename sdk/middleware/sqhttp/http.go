package sqhttp

import (
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
		// Create a new sqreen request wrapper.
		req := sdk.NewHTTPRequest(r)
		defer req.Close()
		// Use the newly created request compliant with `sdk.FromContext()`.
		r = req.Request()

		// Check if a security action is required.
		if handler := req.SecurityAction(); handler != nil {
			handler.ServeHTTP(w, r)
			return
		}

		// Call next handler.
		next.ServeHTTP(w, r)
	})
}

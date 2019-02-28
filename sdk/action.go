package sdk

import "net/http"

type Action interface {
	// Apply the action to the HTTP response writer.
	Apply(w http.ResponseWriter)
}

// SecurityAction returns the security action to perform on the request. It
// returns nil if no action is required. In case of a non-nil action, it should be applied to the request's response writer and should abort the request.
func SecurityAction(req *http.Request) Action {
	return agent.SecurityAction(req)
}

// This package is the contract between the agent and the SDK. It allows to
// strictly separate the SDK from the agent package since the agent does not
// export its internals.
package types

import (
	"encoding/json"
	"net/http"
	"time"
)

type Agent interface {
	// NewRequestRecord returns a new request record for the given request. It
	// should be stored into the request context to be retrieved using
	// `sdk.FromContext()`.
	NewRequestRecord(req *http.Request) RequestRecord

	GracefulStop()
}

type RequestRecord interface {
	// NewCustomEvent creates a new custom event and adds it to the request record.
	NewCustomEvent(event string) CustomEvent
	// NewUserSignup creates a new user signup event and adds it to the request record.
	NewUserSignup(id map[string]string)
	// NewUserAuth creates a new user auth event and adds it to the request record.
	NewUserAuth(id map[string]string, success bool)
	// Identify associates the given user identifiers to the request.
	Identify(id map[string]string)
	// SecurityResponse returns a non-nil HTTP handler when a security response is
	// required for the current request, according to its IP address (taken from
	// the request IP address). The returned handler should be used to respond to
	// the request before canceling it. When a security response matches the
	// request, its value is cached and returned to subsequent calls.
	SecurityResponse() http.Handler
	// UserSecurityResponse returns a non-nil HTTP handler when a security
	// response is required for the current request, according to its
	// user-identifiers (taken from method `Identify()`). The returned handler
	// should be used to respond to the request before canceling it. When a
	// security response matches the request, its value is cached and returned to
	// subsequent calls.
	UserSecurityResponse() http.Handler
	// Close needs to be called when the request is done.
	Close()
}

type CustomEvent interface {
	WithTimestamp(t time.Time)
	WithProperties(props EventProperties)
	WithUserIdentifiers(id map[string]string)
}

// EventProperties is an interface type enforcing a marshable type to the target
// JSON wire-format.
type EventProperties json.Marshaler

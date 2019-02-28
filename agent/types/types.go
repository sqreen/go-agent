// This package is the contract between the agent and the SDK. It allows to
// strictly separate the SDK from the agent package since the agent does not
// export its internals.

package types

import (
	"net/http"
	"time"
)

type Agent interface {
	NewRequestRecord(req *http.Request) RequestRecord
	SecurityAction(req *http.Request) Action
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
	// Close needs to be called when the request is done.
	Close()
}

type CustomEvent interface {
	WithTimestamp(t time.Time)
	WithProperties(props map[string]string)
	WithUserIdentifiers(id map[string]string)
}

type Action interface {
	Apply(w http.ResponseWriter)
}

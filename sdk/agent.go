package sdk

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

// The agent entrypoints are disabled by default. It must set its entrypoints on
// initialization using SetAgent().
var agent Agent = disabledAgent{}

// SetAgent allows the agent to set its SDK entrypoints. It is automatically set
// by the agent when it intializes itself.
func SetAgent(a Agent) {
	if a == nil {
		agent = disabledAgent{}
		return
	}
	agent = a
}

func GracefulStop() {
	agent.GracefulStop()
}

type disabledAgent struct {
}

func (_ disabledAgent) GracefulStop() {
}

func (a disabledAgent) NewRequestRecord(_ *http.Request) RequestRecord {
	// Return itself as long as it can both implement RequestRecord and Agent
	// interfaces without conflicting thanks to distinct method signatures.
	return a
}

func (_ disabledAgent) Close() {
}

func (a disabledAgent) NewCustomEvent(_ string) CustomEvent {
	// Return itself as long as it can both implement RequestRecord and Event
	// interfaces without conflicting thanks to distinct method signatures.
	return a
}

func (_ disabledAgent) NewUserAuth(_ map[string]string, _ bool) {
}

func (_ disabledAgent) NewUserSignup(_ map[string]string) {
}

func (_ disabledAgent) Identify(_ map[string]string) {
}

func (_ disabledAgent) WithTimestamp(_ time.Time) {
}

func (_ disabledAgent) WithProperties(_ map[string]string) {
}

func (_ disabledAgent) WithUserIdentifiers(_ map[string]string) {
}

func (a disabledAgent) SecurityAction(*http.Request) Action {
	return a
}

func (disabledAgent) Apply(http.ResponseWriter) {
}

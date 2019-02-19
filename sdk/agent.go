package sdk

import (
	"net/http"
	"time"

	"github.com/sqreen/go-agent/agent/types"
)

// The agent entrypoints are disabled by default. It must set its entrypoints on
// initialization using SetAgent().
var agent types.Agent = disabledAgent{}

// SetAgent allows the agent to set its SDK entrypoints. It is automatically set
// by the agent when it intializes itself.
func SetAgent(a types.Agent) {
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

func (a disabledAgent) NewRequestRecord(_ *http.Request) types.RequestRecord {
	// Return itself as long as it can both implement RequestRecord and Agent
	// interfaces without conflicting thanks to distinct method signatures.
	return a
}

func (_ disabledAgent) Close() {
}

func (a disabledAgent) NewCustomEvent(_ string) types.CustomEvent {
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

package sdk

import (
	"net/http"

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

func (a disabledAgent) NewRequestRecord(*http.Request) types.RequestRecord {
	return nil
}

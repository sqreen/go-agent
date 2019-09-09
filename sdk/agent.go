// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sdk

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/types"
)

// The agent entry points are disabled by default. It must set its entry points
// on initialization using `SetAgent()`.
var agent types.Agent = disabledAgent{}

// SetAgent allows the agent to set its SDK entry points. It is automatically
// set by the agent when it initializes itself.
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

type disabledAgent struct{}

func (_ disabledAgent) GracefulStop() {}
func (a disabledAgent) NewRequestRecord(r *http.Request) (types.RequestRecord, *http.Request) {
	return nil, r
}

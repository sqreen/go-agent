package agent

import (
	"github.com/sqreen/go-agent/agent/internal"
	"github.com/sqreen/go-agent/sdk"
)

var agent *internal.Agent

func init() {
	agent = internal.New()
	sdk.SetAgent(agent)
	if agent != nil {
		agent.Start()
	}
}

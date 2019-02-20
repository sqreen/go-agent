package agent

import (
	"github.com/sqreen/go-agent/agent/internal"
)

var agent *internal.Agent

func init() {
	agent = internal.New()
	if agent != nil {
		agent.Start()
	}
}

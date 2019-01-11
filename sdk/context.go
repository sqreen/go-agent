package sdk

import (
	"github.com/sqreen/go-agent/agent"
)

type HTTPRequestContext struct {
	ctx *agent.HTTPRequestContext
}

type HTTPRequest = agent.HTTPRequest

func NewHTTPRequestContext(req HTTPRequest) *HTTPRequestContext {
	return &HTTPRequestContext{
		ctx: agent.NewHTTPRequestContext(req),
	}
}

package sdk

import "github.com/sqreen/go-agent/agent"

type HTTPRequestEvent = agent.HTTPRequestEvent

type EventPropertyMap = agent.EventPropertyMap

func (ctx *HTTPRequestContext) Track(event string) *HTTPRequestEvent {
	return ctx.ctx.Track(event)
}

func (ctx *HTTPRequestContext) Close() {
	ctx.ctx.Close()
}

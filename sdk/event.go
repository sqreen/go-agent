package sdk

import (
	"time"

	"github.com/sqreen/go-agent/agent"
)

type HTTPRequestEvent struct {
	impl *agent.HTTPRequestEvent
}

type EventPropertyMap map[string]string

func (e *HTTPRequestEvent) WithTimestamp(t time.Time) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.impl.WithTimestamp(t)
	return e
}

func (e *HTTPRequestEvent) WithProperties(p EventPropertyMap) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.impl.WithProperties(agent.EventPropertyMap(p))
	return e
}

func (e *HTTPRequestEvent) WithUserIdentifier(id EventUserIdentifierMap) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.impl.WithUserIdentifier(agent.EventUserIdentifierMap(id))
	return e
}

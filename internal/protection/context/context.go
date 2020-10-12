// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package context

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

// ContextKey is the context key value to use in order to store a protection
// context into a request context in HTTP middlewares before calling the
// handler.
var ContextKey = contextKey{"_sqreen_ctx_"}

// ContextKey allows to insert context values avoiding string collisions. Cf.
// `context.WithValue()`.
type contextKey struct {
	// This string value must be used by middleware functions whose framework
	// expects context keys of type string, such as Gin. `sdk.FromContext()`
	// expect this behaviour to fallback to string keys when getting the value
	// from the pointer address returned null.
	String string
}

func FromContext(ctx context.Context) interface{} {
	v := ctx.Value(ContextKey)
	if v == nil {
		// Try with a string since frameworks such as Gin implement it with keys of
		// type string.
		v = ctx.Value(ContextKey.String)
	}
	return v
}

// EventRecorderGetter must be implemented by protection contexts so that
// `sdk.FromContext` can get it to instantiate the SDK.
type EventRecorderGetter interface {
	EventRecorder() EventRecorder
}

// EventRecorder is the interface use by the SDK.
type EventRecorder interface {
	// TrackEvent records a new custom event and returns its object to
	// further specify it if required.
	TrackEvent(event string) CustomEvent
	// TrackUserSignup records a new user signup event.
	TrackUserSignup(id map[string]string)
	// TrackUserAuth records a new user authentication (sign-in) event.
	TrackUserAuth(id map[string]string, success bool)
	// IdentifyUser globally associates the given user identifiers to the current
	// request. An non-nil error is returned when a security response matches
	// the given user id.
	IdentifyUser(id map[string]string) error
}

type CustomEvent interface {
	WithTimestamp(t time.Time)
	WithProperties(props EventProperties)
	WithUserIdentifiers(id map[string]string)
}

// EventProperties is an interface type enforcing a marshable type to the target
// JSON wire-format.
type EventProperties json.Marshaler

type AgentFace interface {
	FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error)
	FindActionByUserID(userID map[string]string) (action actor.Action, exists bool)
	Logger() *plog.Logger
	Config() ConfigReader
	// Send the closed request context to the agent. An error is when the object
	// could not be sent (eg. full channel).
	SendClosedRequestContext(ClosedRequestContextFace) error
	IsIPAllowed(ip net.IP) bool
	IsPathAllowed(path string) bool
}

type ClosedRequestContextFace interface{}

type ConfigReader interface {
	PrioritizedIPHeader() string
	PrioritizedIPHeaderFormat() string
}

type ProtectionContext interface {
	AddRequestParam(name string, v interface{})
	HandleAttack(block bool, attack interface{}) (blocked bool)
	ClientIP() net.IP
	SqreenTime() *sqtime.SharedStopWatch
	DeadlineExceeded(d time.Duration) (exceeded bool)
}

func FromGLS() ProtectionContext {
	ctx := sqgls.Get()
	if ctx == nil {
		return nil
	}
	v, _ := ctx.(ProtectionContext)
	return v
}

// Root request context
type RequestContext struct {
	AgentFace
	sqreenTime    sqtime.SharedStopWatch
	maxSqreenTime time.Duration
}

func NewRequestContext(agent AgentFace) *RequestContext {
	return &RequestContext{
		AgentFace: agent,
	}
}

func (c *RequestContext) Close(ctx ClosedRequestContextFace) error {
	return c.AgentFace.SendClosedRequestContext(ctx)
}

func (p *RequestContext) SqreenTime() *sqtime.SharedStopWatch {
	return &p.sqreenTime
}

func (p *RequestContext) DeadlineExceeded(d time.Duration) (exceeded bool) {
	if p.maxSqreenTime == 0 {
		// No max time duration
		return false
	}
	return p.sqreenTime.Duration()+d >= p.maxSqreenTime
}

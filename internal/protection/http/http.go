// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/event"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
)

type ProtectionContext struct {
	context.Context
	types.RootProtectionContext

	RequestReader  types.RequestReader
	ResponseWriter types.ResponseWriter

	events event.Record

	cancelHandlerContextFunc   context.CancelFunc
	contextHandlerCanceledLock sync.RWMutex
	// We are intentionally not using the Context.Err() method here in order to
	// be sure it was canceled by a call to CancelHandlerContext(). Using
	// Context.Err() in order to know this would be also true if for example
	// the parent context timeouts, in which case we mustn't write the blocking
	// response.
	contextHandlerCanceled bool
	requestReader          *requestReader
	start                  time.Time
}

type SecurityResponseStore interface {
	FindActionByIP(ip net.IP) (actor.Action, bool, error)
	FindActionByUserID(id map[string]string) (actor.Action, bool)
}

//func FromContext(ctx context.Context) *ProtectionContext {
//	c, _ := protection_context.FromContext(ctx).(*ProtectionContext)
//	return c
//}

func NewProtectionContext(ctx context.Context, root types.RootProtectionContext, w types.ResponseWriter, r types.RequestReader) *ProtectionContext {
	if root == nil {
		return nil
	}

	if root.IsPathAllowed(r.URL().Path) {
		return nil
	}

	cfg := root.Config()
	clientIP := ClientIP(r.RemoteAddr(), r.Headers(), cfg.HTTPClientIPHeader(), cfg.HTTPClientIPHeaderFormat())

	if root.IsIPAllowed(clientIP) {
		return nil
	}

	reqCtx, cancelHandlerContextFunc := context.WithCancel(ctx)

	rr := &requestReader{
		RequestReader: r,
		clientIP:      clientIP,
		requestParams: make(types.RequestParamMap),
	}

	p := &ProtectionContext{
		Context:                  reqCtx,
		RootProtectionContext:    root,
		ResponseWriter:           w,
		RequestReader:            rr,
		requestReader:            rr,
		cancelHandlerContextFunc: cancelHandlerContextFunc,
	}
	return p
}

// Helper types for callbacks who must be designed for this protection so that
// they are the source of truth and so that the compiler catches type issues
// when compiling (versus when the callback is attached).
type (
	BlockingPrologCallbackType = func(**ProtectionContext) (BlockingEpilogCallbackType, error)
	BlockingEpilogCallbackType = func(*error)

	NonBlockingPrologCallbackType = func(**ProtectionContext) (NonBlockingEpilogCallbackType, error)
	NonBlockingEpilogCallbackType = func()

	WAFPrologCallbackType = BlockingPrologCallbackType
	WAFEpilogCallbackType = BlockingEpilogCallbackType

	BodyWAFPrologCallbackType = WAFPrologCallbackType
	BodyWAFEpilogCallbackType = WAFEpilogCallbackType

	IdentifyUserPrologCallbackType = func(**ProtectionContext, *map[string]string) (BlockingEpilogCallbackType, error)

	ResponseMonitoringPrologCallbackType = func(**ProtectionContext, *types.ResponseFace) (NonBlockingEpilogCallbackType, error)
)

// Static assert that ProtectionContext implements the expected interfaces.
var _ protection_context.EventRecorder = (*ProtectionContext)(nil)

func (p *ProtectionContext) TrackEvent(event string) protection_context.CustomEvent {
	return p.events.AddCustomEvent(event)
}

func (p *ProtectionContext) TrackUserSignup(id map[string]string) {
	p.events.AddUserSignup(id, p.RequestReader.ClientIP())
}

func (p *ProtectionContext) TrackUserAuth(id map[string]string, success bool) {
	p.events.AddUserAuth(id, p.RequestReader.ClientIP(), success)
}

func (p *ProtectionContext) IdentifyUser(id map[string]string) error {
	p.events.Identify(id)
	return p.userSecurityResponse(id)
}

// When a non-nil error is returned, the request handler shouldn't be called
// and the request should be stopped immediately by closing the ProtectionContext
// and returning.
func (p *ProtectionContext) Before() (err error) {
	p.start = time.Now()

	// Set the current goroutine local storage to this request storage to be able
	// to retrieve it from lower-level functions.
	sqgls.Set(p)

	p.addSecurityHeaders()

	if err := p.isIPBlocked(); err != nil {
		return err
	}
	if err := p.ipSecurityResponse(); err != nil {
		return err
	}
	if err := p.waf(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func (p *ProtectionContext) isIPBlocked() error { /* dynamically instrumented */ return nil }

//go:noinline
func (p *ProtectionContext) waf() error { /* dynamically instrumented */ return nil }

//go:noinline
func (p *ProtectionContext) bodyWAF() error { /* dynamically instrumented */ return nil }

//go:noinline
func (p *ProtectionContext) addSecurityHeaders() { /* dynamically instrumented */ }

//go:noinline
func (p *ProtectionContext) ipSecurityResponse() error { /* dynamically instrumented */ return nil }

type canceledHandlerContextError struct{}

func (canceledHandlerContextError) Error() string { return "canceled handler context" }

//go:noinline
func (p *ProtectionContext) After() (err error) {
	if p.isContextHandlerCanceled() {
		// The context was canceled by an in-handler protection, return an error
		// in order to fully abort the framework.
		return canceledHandlerContextError{}
	}

	return nil
}

func (p *ProtectionContext) userSecurityResponse(userID map[string]string) error { return nil }

// CancelHandlerContext cancels the request handler context in order to stop its
// execution and abort every ongoing operation and goroutine it may be doing.
// Since the handler should return at some point, the After() protection method
// will take care of applying the blocking response.
// This method can be called by multiple goroutines simultaneously.
func (p *ProtectionContext) CancelHandlerContext() {
	p.contextHandlerCanceledLock.Lock()
	defer p.contextHandlerCanceledLock.Unlock()
	p.cancelHandlerContextFunc()
	p.contextHandlerCanceled = true
}

func (p *ProtectionContext) isContextHandlerCanceled() bool {
	p.contextHandlerCanceledLock.RLock()
	defer p.contextHandlerCanceledLock.RUnlock()
	return p.contextHandlerCanceled

}

func (p *ProtectionContext) HandleAttack(block bool, attack interface{}) (blocked bool) {
	if block {
		defer p.CancelHandlerContext()
		p.WriteDefaultBlockingResponse()
		blocked = true
	}

	if attack != nil {
		p.events.AddAttackEvent(attack)
	}
	return blocked
}

func (p *ProtectionContext) Close(response types.ResponseFace) {
	p.CancelHandlerContext()

	// Compute the request duration
	duration := time.Since(p.start)

	// Make sure to clear the goroutine local storage to avoid keeping it if some
	// memory pools are used under the hood.
	// TODO: enforce this by design of the gls instrumentation
	defer sqgls.Set(nil)

	// Copy everything we need here as it is not safe to keep then after the
	// request is done because of memory pools reusing them.
	p.monitorObservedResponse(response)
	p.RootProtectionContext.Close(&closedProtectionContext{
		response:   response,
		request:    copyRequest(p.RequestReader),
		events:     p.events.CloseRecord(),
		start:      p.start,
		duration:   duration,
		sqreenTime: p.SqreenTime().Duration(),
	})
}

// Write the default blocking response. This method only write the response, it
// doesn't block nor cancel the handler context. Users of this method must
// handle their
//go:noinline
func (p *ProtectionContext) WriteDefaultBlockingResponse() { /* dynamically instrumented */ }

//go:noinline
func (p *ProtectionContext) monitorObservedResponse(response types.ResponseFace) { /* dynamically instrumented */ }

// WrapRequest is a helper method to prepare an http.Request with its
// new context, the protection context, and a body buffer.
func (p *ProtectionContext) WrapRequest(r *http.Request) *http.Request {
	r = r.WithContext(context.WithValue(p, protection_context.ContextKey, p))
	if r.Body != nil {
		r.Body = p.wrapBody(r.Body)
	}
	return r
}

func (p *ProtectionContext) wrapBody(body io.ReadCloser) io.ReadCloser {
	return rawBodyWAF{
		ReadCloser: body,
		c:          p,
	}
}

// AddRequestParam adds a new request parameter to the set. Request parameters
// are taken from the HTTP request and parsed into a Go value. It can be the
// result of a JSON parsing, query-string parsing, etc. The source allows to
// specify where it was taken from.
func (p *ProtectionContext) AddRequestParam(name string, param interface{}) {
	params := p.requestReader.requestParams[name]
	p.requestReader.requestParams[name] = append(params, param)
}

func (p *ProtectionContext) ClientIP() net.IP {
	return p.requestReader.clientIP
}

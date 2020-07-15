// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/sqreen/go-agent/internal/event"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
)

func FromContext(ctx context.Context) *RequestContext {
	c, _ := protectioncontext.FromContext(ctx).(*RequestContext)
	return c
}

func FromGLS() *RequestContext {
	ctx := sqgls.Get()
	if ctx == nil {
		return nil
	}
	return ctx.(*RequestContext)
}

func NewRequestContext(ctx context.Context, agent protectioncontext.AgentFace, w types.ResponseWriter, r types.RequestReader) (*RequestContext, context.Context, context.CancelFunc) {
	if agent.IsPathAllowed(r.URL().Path) {
		return nil, nil, nil
	}

	clientIP := r.ClientIP()
	if clientIP == nil {
		cfg := agent.Config()
		clientIP = ClientIP(r.RemoteAddr(), r.Headers(), cfg.PrioritizedIPHeader(), cfg.PrioritizedIPHeaderFormat())
	}

	if agent.IsIPAllowed(clientIP) {
		return nil, nil, nil
	}

	reqCtx, cancelHandlerContextFunc := context.WithCancel(ctx)

	rr := &requestReader{
		clientIP:      clientIP,
		RequestReader: r,
		requestParams: make(types.RequestParamMap),
	}

	protCtx := &RequestContext{
		RequestContext:           protectioncontext.NewRequestContext(agent),
		ResponseWriter:           w,
		RequestReader:            rr,
		cancelHandlerContextFunc: cancelHandlerContextFunc,
		// Keep a reference to the request param map to be able to add more params
		// to it.
		requestReader: rr,
	}
	return protCtx, reqCtx, cancelHandlerContextFunc
}

type RequestContext struct {
	*protectioncontext.RequestContext
	RequestReader              types.RequestReader
	ResponseWriter             types.ResponseWriter
	events                     event.Record
	cancelHandlerContextFunc   context.CancelFunc
	contextHandlerCanceledLock sync.RWMutex

	// We are intentionally not using the Context.Err() method here in order to
	// be sure it was canceled by a call to CancelHandlerContext(). Using
	// Context.Err() in order to know this would be also true if for example
	// the parent context timeouts, in which case we mustn't write the blocking
	// response.
	contextHandlerCanceled bool

	requestReader *requestReader
}

// Helper types for callbacks who must be designed for this protection so that
// they are the source of truth and so that the compiler catches type issues
// when compiling (versus when the callback is attached).
type (
	BlockingPrologCallbackType = func(**RequestContext) (BlockingEpilogCallbackType, error)
	BlockingEpilogCallbackType = func(*error)

	NonBlockingPrologCallbackType = func(**RequestContext) (NonBlockingEpilogCallbackType, error)
	NonBlockingEpilogCallbackType = func()

	WAFPrologCallbackType = BlockingPrologCallbackType
	WAFEpilogCallbackType = BlockingEpilogCallbackType

	BodyWAFPrologCallbackType = WAFPrologCallbackType
	BodyWAFEpilogCallbackType = WAFEpilogCallbackType

	IdentifyUserPrologCallbackType = func(**RequestContext, *map[string]string) (BlockingEpilogCallbackType, error)

	ResponseMonitoringPrologCallbackType = func(**RequestContext, *types.ResponseFace) (NonBlockingEpilogCallbackType, error)
)

// Static assert that RequestContext implements the SDK Event Recorder Getter
// interface.
var _ protectioncontext.EventRecorderGetter = (*RequestContext)(nil)

func (c *RequestContext) EventRecorder() protectioncontext.EventRecorder { return c }

func (c *RequestContext) AddAttackEvent(attack *event.AttackEvent) {
	c.events.AddAttackEvent(attack)
}

func (c *RequestContext) TrackEvent(event string) protectioncontext.CustomEvent {
	return c.events.AddCustomEvent(event)
}

func (c *RequestContext) TrackUserSignup(id map[string]string) {
	c.events.AddUserSignup(id, c.RequestReader.ClientIP())
}

func (c *RequestContext) TrackUserAuth(id map[string]string, success bool) {
	c.events.AddUserAuth(id, c.RequestReader.ClientIP(), success)
}

func (c *RequestContext) IdentifyUser(id map[string]string) error {
	c.events.Identify(id)
	return c.userSecurityResponse(id)
}

// Static assert that the SDK interface is implemented.
var _ protectioncontext.EventRecorder = &RequestContext{}

// When a non-nil error is returned, the request handler shouldn't be called
// and the request should be stopped immediately by closing the RequestContext
// and returning.
func (c *RequestContext) Before() (err error) {
	// Set the current goroutine local storage to this request storage to be able
	// to retrieve it from lower-level functions.
	sqgls.Set(c)

	c.addSecurityHeaders()

	if err := c.isIPBlocked(); err != nil {
		return err
	}
	if err := c.ipSecurityResponse(); err != nil {
		return err
	}
	if err := c.waf(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func (c *RequestContext) isIPBlocked() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) waf() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) bodyWAF() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) addSecurityHeaders() { /* dynamically instrumented */ }

//go:noinline
func (c *RequestContext) ipSecurityResponse() error { /* dynamically instrumented */ return nil }

type canceledHandlerContextError struct{}

func (canceledHandlerContextError) Error() string { return "canceled handler context" }

//go:noinline
func (c *RequestContext) After() (err error) {
	if c.isContextHandlerCanceled() {
		// The context was canceled by an in-handler protection, return an error
		// in order to fully abort the framework.
		return canceledHandlerContextError{}
	}

	return nil
}

func (c *RequestContext) userSecurityResponse(userID map[string]string) error { return nil }

// CancelHandlerContext cancels the request handler context in order to stop its
// execution and abort every ongoing operation and goroutine it may be doing.
// Since the handler should return at some point, the After() protection method
// will take care of applying the blocking response.
// This method can be called by multiple goroutines simultaneously.
func (c *RequestContext) CancelHandlerContext() {
	c.contextHandlerCanceledLock.Lock()
	defer c.contextHandlerCanceledLock.Unlock()
	c.contextHandlerCanceled = true
}

func (c *RequestContext) isContextHandlerCanceled() bool {
	c.contextHandlerCanceledLock.RLock()
	defer c.contextHandlerCanceledLock.RUnlock()
	return c.contextHandlerCanceled

}

func (c *RequestContext) Close(response types.ResponseFace) error {
	// Make sure to clear the goroutine local storage to avoid keeping it if some
	// memory pools are used under the hood.
	// TODO: enforce this by design of the gls instrumentation
	defer sqgls.Set(nil)

	// Copy everything we need here as it is not safe to keep then after the
	// request is done because of memory pools reusing them.
	c.monitorObservedResponse(response)
	return c.RequestContext.Close(&closedRequestContext{
		response: response,
		request:  copyRequest(c.RequestReader),
		events:   c.events.CloseRecord(),
	})
}

// Write the default blocking response. This method only write the response, it
// doesn't block nor cancel the handler context. Users of this method must
// handle their
//go:noinline
func (c *RequestContext) WriteDefaultBlockingResponse() { /* dynamically instrumented */ }

//go:noinline
func (c *RequestContext) monitorObservedResponse(response types.ResponseFace) { /* dynamically instrumented */ }

// WrapRequest is a helper method to prepare an http.Request with its
// new context, the protection context, and a body buffer.
func (c *RequestContext) WrapRequest(ctx context.Context, r *http.Request) *http.Request {
	r = r.WithContext(context.WithValue(ctx, protectioncontext.ContextKey, c))
	if r.Body != nil {
		r.Body = c.wrapBody(r.Body)
	}
	return r
}

func (c *RequestContext) wrapBody(body io.ReadCloser) io.ReadCloser {
	return rawBodyWAF{
		ReadCloser: body,
		c:          c,
	}
}

// AddRequestParam adds a new request parameter to the set. Request parameters
// are taken from the HTTP request and parsed into a Go value. It can be the
// result of a JSON parsing, query-string parsing, etc. The source allows to
// specify where it was taken from.
func (c *RequestContext) AddRequestParam(name string, param interface{}) {
	params := c.requestReader.requestParams[name]
	c.requestReader.requestParams[name] = append(params, param)
}

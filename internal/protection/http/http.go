// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/event"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/span"
)

type ProtectionContext struct {
	types.RootProtectionContext

	RequestReader  types.RequestReader
	ResponseWriter types.ResponseWriter

	events event.Record

	requestBindingAccessorFeed *requestBindingAccessorFeed
	start                      time.Time
}

type SecurityResponseStore interface {
	FindActionByIP(ip net.IP) (actor.Action, bool, error)
	FindActionByUserID(id map[string]string) (actor.Action, bool)
}

func NewProtectionContext(ctx types.RootProtectionContext, w types.ResponseWriter, r types.RequestReader) *ProtectionContext {
	if ctx == nil {
		return nil
	}

	if ctx.IsPathAllowed(r.URL().Path) {
		return nil
	}

	cfg := ctx.Config()
	clientIP := ClientIP(r.RemoteAddr(), r.Headers(), cfg.HTTPClientIPHeader(), cfg.HTTPClientIPHeaderFormat())

	if ctx.IsIPAllowed(clientIP) {
		return nil
	}

	rr := &requestBindingAccessorFeed{
		RequestReader: r,
		clientIP:      clientIP,
	}

	p := &ProtectionContext{
		RootProtectionContext:      ctx,
		ResponseWriter:             w,
		RequestReader:              r,
		requestBindingAccessorFeed: rr,
	}

	return p
}

func NewTestProtectionContext(ctx types.RootProtectionContext, clientIP net.IP, w types.ResponseWriter, r types.RequestReader) *ProtectionContext {
	rr := &requestBindingAccessorFeed{
		RequestReader: r,
		clientIP:      clientIP,
	}

	return &ProtectionContext{
		RootProtectionContext:      ctx,
		ResponseWriter:             w,
		RequestReader:              rr,
		requestBindingAccessorFeed: rr,
	}
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
	p.events.AddUserSignup(id, p.ClientIP())
}

func (p *ProtectionContext) TrackUserAuth(id map[string]string, success bool) {
	p.events.AddUserAuth(id, p.ClientIP(), success)
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

	p.addSecurityHeaders()

	// TODO: turn IP-based protections into span callbacks
	if err := p.isIPBlocked(); err != nil {
		return err
	}
	if err := p.ipSecurityResponse(); err != nil {
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

func (p *ProtectionContext) isContextHandlerCanceled() bool {
	return p.Context().Err() == context.Canceled

}

func (p *ProtectionContext) HandleAttack(block bool, attack *event.AttackEvent) (blocked bool) {
	if block {
		defer p.CancelContext()
		p.WriteDefaultBlockingResponse()
		blocked = true
	}

	if attack != nil {
		p.events.AddAttackEvent(attack)
	}

	return blocked
}

func (p *ProtectionContext) Close(response types.ResponseFace) {
	// Compute the request duration
	duration := time.Since(p.start)

	// Copy everything we need here as it is not safe to keep then after the
	// request is done because of memory pools reusing them.
	p.RootProtectionContext.Close(&closedProtectionContext{
		response:   response,
		request:    copyRequest(p.requestBindingAccessorFeed, p.ClientIP()),
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

func (p *ProtectionContext) MonitorObservedResponse(response types.ResponseFace) {
	p.monitorObservedResponse(response)
}

//go:noinline
func (p *ProtectionContext) monitorObservedResponse(response types.ResponseFace) {
	/* dynamically instrumented */
}

// WrapRequest is a helper method to prepare an http.Request with its
// new context, the protection context, and a body buffer.
func (p *ProtectionContext) WrapRequest(r *http.Request) *http.Request {
	// TODO: we no longer need to store the full protection context as this is for
	//  sdk events only now - we could just store the event recorder, or
	//  completely rely on the GLS now
	ctx := context.WithValue(r.Context(), protection_context.ContextKey, p)
	r = r.WithContext(ctx)
	return r
}

// SetRequestBindingAccessorValue adds a new request parameter to the set of
// request parameters exposed to binding accessors.
// Examples include the result of a JSON parsing of the request body,
// query-string parsing, etc.
// The address is the span address used to publish this value.
func (p *ProtectionContext) SetRequestBindingAccessorValue(address string, value interface{}) {
	p.requestBindingAccessorFeed.params.Set(address, value)
}

func (p *ProtectionContext) ClientIP() net.IP {
	return p.requestBindingAccessorFeed.clientIP
}

func (p *ProtectionContext) RequestBindingAccessorReader() types.RequestBindingAccessorReader {
	return p.requestBindingAccessorFeed
}

func NewHTTPHandlerSpan(ip net.IP, r types.RequestReader) (span.SpanEnder, error) {
	// TODO: request reader function to span attributes + tests with raw
	//   unescaped values
	headers := make(map[string][]string, len(r.Headers()))
	var cookies []string
	for h, v := range r.Headers() {
		if h == "Cookie" {
			cookies = v
			continue
		}
		// TODO: have a custom waf marshaler instead so that we can avoid the string
		//   copy here and rather lower-case the C copy
		headers[strings.ToLower(h)] = v
	}

	attrs := span.AttributeMap{
		"span.name":                         "http.handler",
		"server.request.headers.no_cookies": headers,
		"server.request.uri.raw":            r.RequestURI(),
		"server.request.client_ip":          ip.String(),
		"server.request.method":             r.Method(),
		"server.request.transport":          r.Transport(),

		// Extra values for our own RE logic
		"server.request.go_url": r.URL(),
	}

	if len(cookies) > 0 {
		attrs["server.request.cookies"] = cookies
	}

	if r, ok := r.(types.RequestPathParamsGetter); ok {
		attrs["server.request.path_params"] = r.PathParams()
	}

	return span.NewSpan(span.WithAttributes(attrs))
}

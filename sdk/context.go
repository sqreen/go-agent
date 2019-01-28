package sdk

import (
	"context"

	"github.com/sqreen/go-agent/agent"
)

const HTTPRequestContextKey = "sqctx"

// HTTPRequestContext is the context associated to a single HTTP request. It
// collects every security event happening during the HTTP handling, until the
// Close() method is called to signal the request is done.
type HTTPRequestContext struct {
	ctx *agent.HTTPRequestContext
}

// GetHTTPContext returns the sdk's context associated to the request's context
// by the middleware function.
//
//	router.GET("/", func(c *gin.Context) {
//		sqgin.GetHTTPContext(c).TrackEvent("my.event.one")
//		aFunction(c.Request.Context())
//	}
//
//	func aFunction(ctx context.Context) {
//		sqgin.GetHTTPContext(ctx).TrackEvent("my.event.two")
//		// ...
//	}
//
func GetHTTPContext(ctx context.Context) *HTTPRequestContext {
	sqreen := ctx.Value(HTTPRequestContextKey)
	if sqreen == nil {
		return nil
	}
	return sqreen.(*HTTPRequestContext)
}

// HTTPRequest is the request interface to access request information. Usually
// implemented by the middleware to give access to the handler's HTTP request.
type HTTPRequest = agent.HTTPRequest

type HTTPRequestEvent = agent.HTTPRequestEvent

type EventPropertyMap = agent.EventPropertyMap

type EventUserIdentifierMap = agent.EventUserIdentifierMap

// NewHTTPRequestContext returns a new HTTP request context for the given HTTP
// request.
func NewHTTPRequestContext(req HTTPRequest) *HTTPRequestContext {
	return &HTTPRequestContext{
		ctx: agent.NewHTTPRequestContext(req),
	}
}

// Close signals the request handling is now done. Every collected security
// event can thus be considered by the agnt.
func (ctx *HTTPRequestContext) Close() {
	if ctx == nil {
		return
	}
	ctx.ctx.Close()
}

// Track allows to track a custom security-related event with the given event
// name. Additional options can be set using the returned value's methods, such
// WithProperties() or WithTimestamp().
//
//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqreen := middleware.GetHTTPContext(ctx)
//	sqreen.TrackEvent("my.event").WithUserIdentifier(uid).WithProperties(props)
//
func (ctx *HTTPRequestContext) TrackEvent(event string) *HTTPRequestEvent {
	if ctx == nil {
		return nil
	}
	return (ctx.ctx.Track(event))
}

// TrackAuth allows to track a user authentication. The user id `id` is a set
// uniquely identifying the user. `loginSuccess` must be true when the user
// successfully logged in, false otherwise.
//
//	sqreen := middleware.GetHTTPContext(ctx)
//	sqreen.TrackAuth(granted, sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestContext) TrackAuth(loginSuccess bool, id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackAuth(loginSuccess, id)
}

// TrackAuthSuccess is equivalent to `TrackAuth(true, id)`
func (ctx *HTTPRequestContext) TrackAuthSuccess(id EventUserIdentifierMap) {
	ctx.TrackAuth(true, id)
}

// TrackAuthFailure is equivalent to `TrackAuth(false, id)`
func (ctx *HTTPRequestContext) TrackAuthFailure(id EventUserIdentifierMap) {
	ctx.TrackAuth(false, id)
}

// TrackSignup allows to track a user signup. The user id `id` is a set
// uniquely identifying the user.
//
//	sqreen := middleware.GetHTTPContext(ctx)
//	sqreen.TrackSignup(sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestContext) TrackSignup(id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackSignup(id)
}

package sdk

import (
	"context"
	"net/http"

	"github.com/sqreen/go-agent/agent"
)

// HTTPRequestRecord is the context associated to a single HTTP request. It
// collects every security event happening during the HTTP handling, until the
// Close() method is called to signal the request is done.
type HTTPRequestRecord struct {
	ctx *agent.HTTPRequestRecord
}

type EventUserIdentifierMap map[string]string

// NewHTTPRequestRecord returns a new HTTP request context for the given HTTP
// request.
func NewHTTPRequestRecord(req *http.Request) *HTTPRequestRecord {
	return &HTTPRequestRecord{
		ctx: agent.NewHTTPRequestRecord(req),
	}
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
func GetHTTPContext(ctx context.Context) *HTTPRequestRecord {
	v := ctx.Value(HTTPRequestRecordContextKey)
	if v == nil {
		// Try with a string since frameworks such as Gin implement it with keys of
		// type string.
		v = ctx.Value(HTTPRequestRecordContextKey.String)
		if v == nil {
			return nil
		}
	}
	return v.(*HTTPRequestRecord)
}

// Close signals the request handling is now done. Every collected security
// event can thus be considered by the agnt.
func (ctx *HTTPRequestRecord) Close() {
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
func (ctx *HTTPRequestRecord) TrackEvent(event string) *HTTPRequestEvent {
	if ctx == nil {
		return nil
	}
	return &HTTPRequestEvent{ctx.ctx.Track(event)}
}

// TrackAuth allows to track a user authentication. The user id `id` is a set
// uniquely identifying the user. `loginSuccess` must be true when the user
// successfully logged in, false otherwise.
//
//	sqreen := middleware.GetHTTPContext(ctx)
//	sqreen.TrackAuth(granted, sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestRecord) TrackAuth(loginSuccess bool, id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackAuth(loginSuccess, agent.EventUserIdentifierMap(id))
}

// TrackAuthSuccess is equivalent to `TrackAuth(true, id)`
func (ctx *HTTPRequestRecord) TrackAuthSuccess(id EventUserIdentifierMap) {
	ctx.TrackAuth(true, id)
}

// TrackAuthFailure is equivalent to `TrackAuth(false, id)`
func (ctx *HTTPRequestRecord) TrackAuthFailure(id EventUserIdentifierMap) {
	ctx.TrackAuth(false, id)
}

// TrackSignup allows to track a user signup. The user id `id` is a set
// uniquely identifying the user.
//
//	sqreen := middleware.GetHTTPContext(ctx)
//	sqreen.TrackSignup(sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestRecord) TrackSignup(id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackSignup(agent.EventUserIdentifierMap(id))
}

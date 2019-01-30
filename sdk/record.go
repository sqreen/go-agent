package sdk

import (
	"context"
	"net/http"

	"github.com/sqreen/go-agent/agent"
)

// HTTPRequestRecord is the SDK record associated to a HTTP request. Its methods
// allow request handlers to signal security events, identify users, etc.
type HTTPRequestRecord struct {
	ctx *agent.HTTPRequestRecord
}

// EventUserIdentifierMap is the type used to represent user identifiers in
// collected events.
//
//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
//	sdk.FromContext(ctx).Identify(uid)
//
type EventUserIdentifierMap map[string]string

// NewHTTPRequestRecord returns a new HTTP request record for the given HTTP
// request.
func NewHTTPRequestRecord(req *http.Request) *HTTPRequestRecord {
	return &HTTPRequestRecord{
		ctx: agent.NewHTTPRequestRecord(req),
	}
}

// FromContext allows to access the HTTPRequestRecord from request handlers if
// present, and nil otherwise. The value is stored in handler contexts by the
// middleware function of the framework, and is of type *HTTPRequestRecord. It
// is possible to use it with framework's contexts when they implement Go's
// `context.Context` interface.
//
//	router.GET("/", func(c *gin.Context) {
//		// Accessing the SDK through framework's context (when possible).
//		sdk.FromContext(c).TrackEvent("my.event.one")
//		aFunction(c.Request)
//	}
//
//	func aFunction(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//		// ...
//	}
//
func FromContext(ctx context.Context) *HTTPRequestRecord {
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

// Close the request record to signal the HTTP request handling is now done.
func (ctx *HTTPRequestRecord) Close() {
	if ctx == nil {
		return
	}
	ctx.ctx.Close()
}

// TrackEvent allows to track a custom security-related event having the given
// event name. It creates a new event whose additional options can be set using
// the returned value's methods, such as `WithProperties()` or
// `WithTimestamp()`. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqreen := sdk.FromContext(ctx)
//	sqreen.TrackEvent("my.event").WithUserIdentifier(uid).WithProperties(props)
//
func (ctx *HTTPRequestRecord) TrackEvent(event string) *HTTPRequestEvent {
	if ctx == nil {
		return nil
	}
	return &HTTPRequestEvent{ctx.ctx.Track(event)}
}

// TrackAuth allows to track a user authentication. The given user identifier
// value `id` is a map uniquely identifying the user. The boolean value
// `loginSuccess` must be true when the user successfully logged in, false
// otherwise. A call to this method creates a new event.
//
//	sqreen := sdk.FromContext(ctx)
//	sqreen.TrackAuth(granted, sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestRecord) TrackAuth(loginSuccess bool, id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackAuth(loginSuccess, agent.EventUserIdentifierMap(id))
}

// TrackAuthSuccess is equivalent to `TrackAuth(true, id)`.
func (ctx *HTTPRequestRecord) TrackAuthSuccess(id EventUserIdentifierMap) {
	ctx.TrackAuth(true, id)
}

// TrackAuthFailure is equivalent to `TrackAuth(false, id)`.
func (ctx *HTTPRequestRecord) TrackAuthFailure(id EventUserIdentifierMap) {
	ctx.TrackAuth(false, id)
}

// TrackSignup allows to track a user signup. The given user identifier value
// `id` is a map uniquely identifying the user. A call to this method creates a
// new event.
//
//	sqreen := sdk.FromContext(ctx)
//	sqreen.TrackSignup(sdk.EventUserIdentifierMap{"uid": "my-uid"})
//
func (ctx *HTTPRequestRecord) TrackSignup(id EventUserIdentifierMap) {
	if ctx == nil {
		return
	}
	ctx.ctx.TrackSignup(agent.EventUserIdentifierMap(id))
}

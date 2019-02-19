package sdk

import (
	"context"
	"net/http"

	"github.com/sqreen/go-agent/agent/types"
)

// HTTPRequestRecord is the SDK record associated to a HTTP request. Its methods
// allow request handlers to track custom security events.
type HTTPRequestRecord struct {
	record types.RequestRecord
}

// EventUserIdentifiersMap is the type used to represent user identifiers in
// collected events. It is a key-value map that should uniquely identify a user.
//
// For example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sdk.FromContext(ctx).ForUser(uid).TrackEvent("my.event")
//
type EventUserIdentifiersMap map[string]string

// NewHTTPRequestRecord returns a new HTTP request record for the given HTTP
// request.
func NewHTTPRequestRecord(req *http.Request) *HTTPRequestRecord {
	return &HTTPRequestRecord{
		record: agent.NewRequestRecord(req),
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
	ctx.record.Close()
}

// TrackEvent allows to track a custom security-related event having the given
// event name. It creates a new event whose additional options can be set using
// the returned value's methods, such as `WithProperties()` or
// `WithTimestamp()`. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqreen := sdk.FromContext(ctx)
//	sqreen.TrackEvent("my.event").WithUserIdentifiers(uid).WithProperties(props)
//
func (ctx *HTTPRequestRecord) TrackEvent(event string) *HTTPRequestEvent {
	if ctx == nil {
		return nil
	}
	return &HTTPRequestEvent{ctx.record.NewCustomEvent(event)}
}

// ForUser returns a new SDK context for the given user uniquely identified by
// `id`. Its methods allow to track security events related to this user. A call
// to this method does not create a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackAuthSuccess()
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqUser.TrackEvent("my.event.one").WithProperties(props)
//
func (ctx *HTTPRequestRecord) ForUser(id EventUserIdentifiersMap) *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	return &UserHTTPRequestRecord{
		record: ctx.record,
		id:     id,
	}
}

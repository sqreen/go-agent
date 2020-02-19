// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package sdk

import (
	"context"
	"encoding/json"
	"time"

	protection_context "github.com/sqreen/go-agent/internal/protection/context"
)

// Deprecated: type name of Context.
type HTTPRequestRecord = Context

// TODO: doc
type Context struct {
	events protection_context.EventRecorder
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

// FromContext allows to access the request record from request handlers if
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
func FromContext(ctx context.Context) *Context {
	v := protection_context.FromContext(ctx)
	if v == nil {
		return &Context{events: disabledEventRecorder{}}
	}
	c, ok := v.(protection_context.EventRecorderGetter)
	if !ok {
		return &Context{events: disabledEventRecorder{}}
	}
	return &Context{events: c.EventRecorder()}
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
func (ctx *Context) TrackEvent(event string) *TrackEvent {
	return &TrackEvent{event: ctx.events.TrackEvent(event)}
}

// EventPropertyMap is the type used to represent extra custom event properties.
//
//	props := sdk.EventPropertyMap{
//		"key1": "value1",
//		"key2": "value2",
//	}
//	sdk.FromContext(ctx).TrackEvent("my.event").WithProperties(props)
//
type EventPropertyMap map[string]string

func (m EventPropertyMap) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]string(m)) }

// TrackEvent is a SDK event. Its methods allow request handlers to add
// options further specifying the event, such as a unique user identifier, extra
// properties, etc.
type TrackEvent struct {
	event protection_context.CustomEvent
}

// HTTPRequestEvent is the deprecated type name of TrackEvent.
type HTTPRequestEvent = TrackEvent

// WithTimestamp adds a custom timestamp to the event. By default, the timestamp
// is set to `time.Now()` value at the time of the call to the event creation.
//
//	sdk.FromContext(ctx).TrackEvent("my.event").WithTimestamp(yourTimestamp)
//
func (e *TrackEvent) WithTimestamp(t time.Time) *TrackEvent {
	e.event.WithTimestamp(t)
	return e
}

// WithProperties adds custom properties to the event.
//
//	props := sdk.EventPropertyMap{
//		"key1": "value1",
//		"key2": "value2",
//	}
//	sdk.FromContext(ctx).TrackEvent("my.event").WithProperties(prop)
//
func (e *TrackEvent) WithProperties(p EventPropertyMap) *TrackEvent {
	e.event.WithProperties(p)
	return e
}

// WithUserIdentifier associates the given user identifier map `id` to the
// event.
//
//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
//	sdk.FromContext(ctx).Identify(uid)
//
func (e *TrackEvent) WithUserIdentifiers(id EventUserIdentifiersMap) *TrackEvent {
	e.event.WithUserIdentifiers(id)
	return e
}

// UserHTTPRequestRecord is the SDK record associated to a HTTP request for a
// given user. Its methods allow request handlers to signal security events
// related to the given user. It allows to send security events related to a
// single user.
type UserContext struct {
	ctx Context
	id  EventUserIdentifiersMap
}

// UserHTTPRequestRecord is the deprecated type name of UserContext.
type UserHTTPRequestRecord = UserContext

// ForUser returns a new user request record for the given user `id`. Its
// methods allow to perform security events related to this user. A call to
// this method does not create a new event.
//
// Note that it doesn't associate the user to the request unless `Identify()`
// is explicitly called.
//
// Usage example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackAuthSuccess()
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqUser.TrackEvent("my.event.one").WithProperties(props)
//
func (ctx Context) ForUser(id EventUserIdentifiersMap) *UserContext {
	// TODO: we can likely return a value instead by changing the method
	//   receivers below to values instead of pointers.
	return &UserContext{
		ctx: ctx,
		id:  id,
	}
}

// TrackAuth allows to track a user authentication. The boolean value
// `loginSuccess` must be true when the user successfully logged in, false
// otherwise. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackAuthSuccess()
//
func (u *UserContext) TrackAuth(loginSuccess bool) *UserContext {
	u.ctx.events.TrackUserAuth(u.id, loginSuccess)
	return u
}

// TrackAuthSuccess is equivalent to `TrackAuth(true)`.
func (u *UserContext) TrackAuthSuccess() *UserContext {
	return u.TrackAuth(true)
}

// TrackAuthFailure is equivalent to `TrackAuth(false)`.
func (u *UserContext) TrackAuthFailure() *UserContext {
	return u.TrackAuth(false)
}

// TrackSignup allows to track a user signup. A call to this method creates a
// new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackSignup()
//
func (u *UserContext) TrackSignup() *UserContext {
	u.ctx.events.TrackUserSignup(u.id)
	return u
}

// TrackEvent is a convenience method to send a custom security event
// associated to the user. It is equivalent to using method
// `WithUserIdentifiers()` of the regular `TrackEvent()` method.
// So it is equivalent to
// `sdk.FromContext(ctx).TrackEvent("event").WithUserIdentifiers(uid)`.
// This alternative should be considered when performing multiple user events
// as it allow to write a few less code.
//
// Usage example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackSignup()
//	if match, _ := sqUser.MatchSecurityResponse(); match {
//		return
//	}
//	sqUser.TrackEvent("my.event.one")
//	sqUser.TrackEvent("my.event.two")
//	// ...
//
func (u *UserContext) TrackEvent(event string) *UserEvent {
	uevent := u.ctx.TrackEvent(event).WithUserIdentifiers(u.id)
	return (*UserEvent)(uevent)
}

// UserEvent is a user monitoring event. Its methods allow request handlers to
// add options further specifying the event, such as a unique user identifier,
// extra properties, etc.
type UserEvent TrackEvent

// UserHTTPRequestEvent is the deprecated type name of UserEvent.
// Deprecated: use UserEvent type name instead.
type UserHTTPRequestEvent = UserEvent

func (e *UserEvent) unwrap() *TrackEvent { return (*TrackEvent)(e) }

// WithTimestamp adds a custom timestamp to the event. By default, the timestamp
// is set to `time.Now()` value at the time of the call to the event creation.
//
//	sdk.FromContext(ctx).TrackEvent("my.event").WithTimestamp(yourTimestamp)
//
func (e *UserEvent) WithTimestamp(t time.Time) *UserEvent {
	e.unwrap().WithTimestamp(t)
	return e
}

// WithProperties adds custom properties to the event.
//
//	props := sdk.EventPropertyMap{
//		"key1": "value1",
//		"key2": "value2",
//	}
//	sdk.FromContext(ctx).TrackEvent("my.event").WithProperties(prop)
//
func (e *UserEvent) WithProperties(p EventPropertyMap) *UserEvent {
	e.unwrap().WithProperties(p)
	return e
}

// Identify globally associates the given user-identifiers to the current
// request. Every event of the request will be automatically associated to the
// user, unless forced using `WithUserIdentifiers()`.
// A non-nil error is returned when a user security response was found and that
// the request should be aborted. The handler response writer is closed when an
// error is returned in order to prevent from responding to the request.
// The request will be blocked according to your settings (blocking page or
// HTTP redirection).
//
// Usage example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	if err := sqUser.Identify(); err != nil {
//		// Return now to stop further handling the request and let Sqreen's
//		// middleware apply the configured security response and abort the
//		// request. Returning the error may help bubbling up the handler call
//	  // stack.
//		return err
//	}
//
func (u *UserContext) Identify() error {
	err := u.ctx.events.IdentifyUser(u.id)
	return err
}

type disabledEventRecorder struct{}

func (disabledEventRecorder) WithTimestamp(time.Time) {}

func (disabledEventRecorder) WithProperties(protection_context.EventProperties) {}

func (disabledEventRecorder) WithUserIdentifiers(map[string]string) {}

func (d disabledEventRecorder) TrackEvent(string) protection_context.CustomEvent { return d }

func (disabledEventRecorder) TrackUserSignup(map[string]string) {}

func (disabledEventRecorder) TrackUserAuth(map[string]string, bool) {}

func (disabledEventRecorder) IdentifyUser(map[string]string) error { return nil }

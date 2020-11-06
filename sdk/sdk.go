// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package sdk

import (
	go_context "context"
	"encoding/json"
	"time"

	protection_context "github.com/sqreen/go-agent/internal/protection/context"
)

type (
	// Context is Sqreen's request context associated to a HTTP request by the
	// middleware function. Its methods allow request handlers to record security
	// events and monitor the user activity.
	Context interface {
		// ForUser returns a new user request context for the given user `id`. Its
		// methods allow to perform security events related to this user. A call to
		// this method does not create a new event but only returns a user handle to
		// perform user events.
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
		//	sqUser.TrackEvent("my.user.event").WithProperties(props)
		//
		ForUser(userID EventUserIdentifiersMap) UserContext

		// TrackEvent allows to track a custom security events with the given event name.
		// It creates a new event whose additional options can be set using the
		// returned value's methods, such as `WithProperties()` or
		// `WithTimestamp()`. A call to this method creates a new event.
		//
		//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
		//	props := sdk.EventPropertyMap{"key": "value"}
		//	sqreen := sdk.FromContext(ctx)
		//	sqreen.TrackEvent("my.event").WithUserIdentifiers(uid).WithProperties(props)
		//
		TrackEvent(name string) TrackEvent
	}

	context struct {
		events protection_context.EventRecorder
	}
)

// EventUserIdentifiersMap is the type used to represent user identifiers in
// collected events. It is a key-value map that should uniquely identify a user.
//
// For example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sdk.FromContext(ctx).ForUser(uid).TrackEvent("my.event")
//
type EventUserIdentifiersMap map[string]string

// FromContext retrieves Sqreen's request context set by the middleware
// function from the given Go request context. If Sqreen is disabled or no
// middleware function is set, it returns a disabled context that will ignore
// everything.
//
// Usage examples:
//
//  // A gin handler function
//	func(c *gin.Context) {
//		// Accessing the SDK through gin framework context
//		sdk.FromContext(c).TrackEvent("my.event.one")
//    // ...
//	}
//
//  // A net/http handler function
//	func handler(w http.ResponseWriter, r *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(r.Context()).TrackEvent("my.event.two")
//		// ...
//	}
//
func FromContext(ctx go_context.Context) Context {
	if ctx == nil {
		return context{events: disabledEventRecorder{}}
	}

	v := ctx.Value(protection_context.ContextKey)
	if v == nil {
		// Try with a string since frameworks such as Gin implement it with keys of
		// type string.
		v = ctx.Value(protection_context.ContextKey.String)
	}

	actual, ok := v.(protection_context.EventRecorder)

	if !ok || actual == nil {
		return context{events: disabledEventRecorder{}}
	}

	return context{events: actual}
}

// TrackEvent allows to track a custom security events with the given event name.
// It creates a new event whose additional options can be set using the
// returned value's methods, such as `WithProperties()` or
// `WithTimestamp()`. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	props := sdk.EventPropertyMap{"key": "value"}
//	sqreen := sdk.FromContext(ctx)
//	sqreen.TrackEvent("my.event").WithUserIdentifiers(uid).WithProperties(props)
//
func (ctx context) TrackEvent(event string) TrackEvent {
	return trackEvent{event: ctx.events.TrackEvent(event)}
}

// EventPropertyMap is the type used to represent extra event properties.
//
//	props := sdk.EventPropertyMap{
//		"key1": "value1",
//		"key2": "value2",
//	}
//	sdk.FromContext(ctx).TrackEvent("my.event").WithProperties(props)
//
type EventPropertyMap map[string]string

func (m EventPropertyMap) MarshalJSON() ([]byte, error) { return json.Marshal(map[string]string(m)) }

type (
	// TrackEvent is a custom security event. Its methods allow to further
	// define the event, such as a unique user identifier or extra properties.
	TrackEvent interface {
		// WithUserIdentifier associates the given user identifier map `id` to the
		// event.
		//
		//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
		//	sdk.FromContext(ctx).Identify(uid)
		//
		WithUserIdentifiers(userID EventUserIdentifiersMap) TrackEvent

		// WithProperties adds custom properties to the event.
		//
		//	props := sdk.EventPropertyMap{
		//		"key1": "value1",
		//		"key2": "value2",
		//	}
		//	sdk.FromContext(ctx).TrackEvent("my.event").WithProperties(prop)
		//
		WithProperties(properties EventPropertyMap) TrackEvent

		// WithTimestamp adds a custom timestamp to the event. By default, the timestamp
		// is set to `time.Now()` value at the time of the call to the event creation.
		//
		//	sdk.FromContext(ctx).TrackEvent("my.event").WithTimestamp(yourTimestamp)
		//
		WithTimestamp(timestamp time.Time) TrackEvent
	}

	trackEvent struct {
		event protection_context.CustomEvent
	}
)

// WithTimestamp adds a custom timestamp to the event. By default, the timestamp
// is set to `time.Now()` value at the time of the call to the event creation.
//
//	sdk.FromContext(ctx).TrackEvent("my.event").WithTimestamp(yourTimestamp)
//
func (e trackEvent) WithTimestamp(t time.Time) TrackEvent {
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
func (e trackEvent) WithProperties(p EventPropertyMap) TrackEvent {
	e.event.WithProperties(p)
	return e
}

// WithUserIdentifier associates the given user identifier map `id` to the
// event.
//
//	uid := sdk.EventUserIdentifierMap{"uid": "my-uid"}
//	sdk.FromContext(ctx).Identify(uid)
//
func (e trackEvent) WithUserIdentifiers(id EventUserIdentifiersMap) TrackEvent {
	e.event.WithUserIdentifiers(id)
	return e
}

type (
	// UserContext is a SDK handle for a given user and current request.
	// Its methods allow request handlers to monitor user activity (login, signup,
	// or identification) or create custom user security events.
	UserContext interface {
		// TrackEvent is a convenience method to send a custom security event
		// associated to the user. It is equivalent to using method
		// `WithUserIdentifiers()` on the regular `TrackEvent()` method.
		// So it is equivalent to
		// `sdk.FromContext(ctx).TrackEvent("event").WithUserIdentifiers(uid)`.
		// This alternative should be considered when performing multiple user events
		// as it allows to write fewer lines.
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
		TrackEvent(name string) UserEvent

		// TrackSignup allows to track a user signup. A call to this method creates a
		// new event.
		//
		//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
		//	sqUser := sdk.FromContext(ctx).ForUser(uid)
		//	sqUser.TrackSignup()
		//
		TrackSignup() UserContext

		// TrackAuth allows to track a user authentication. The boolean value
		// `loginSuccess` must be true when the user successfully logged in, false
		// otherwise. A call to this method creates a new event.
		//
		//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
		//	sqUser := sdk.FromContext(ctx).ForUser(uid)
		//	sqUser.TrackAuthSuccess()
		//
		TrackAuth(success bool) UserContext

		// TrackAuthSuccess is equivalent to `TrackAuth(true)`.
		TrackAuthSuccess() UserContext

		// TrackAuthFailure is equivalent to `TrackAuth(false)`.
		TrackAuthFailure() UserContext

		// Identify globally associates the given UserContext identifiers to the current
		// request and returns a non-nil error if the user was blocked by Sqreen. Note
		// that when an error is returned, the request was already answered with your
		// blocking configuration and the request context was canceled in order to abort
		// every ongoing operation. So the caller shouldn't continue handling the
		// request any further.
		//
		// Every event following this one will be automatically associated to this
		// user, unless forced using `WithUserIdentifiers()`.
		//
		// Usage example:
		//
		//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
		//	sqUser := sdk.FromContext(ctx).ForUser(uid)
		//	if err := sqUser.Identify(); err != nil {
		//		// Return now to stop further handling the request. Returning the error
		//		// may help bubbling up the handler call stack.
		//		return err
		//	}
		//
		Identify() error
	}

	userContext struct {
		ctx context
		id  EventUserIdentifiersMap
	}
)

// ForUser returns a new user request context for the given user `id`. Its
// methods allow to perform security events related to this user. A call to
// this method does not create a new event but only returns a user handle to
// perform user events.
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
//	sqUser.TrackEvent("my.user.event").WithProperties(props)
//
func (ctx context) ForUser(id EventUserIdentifiersMap) UserContext {
	return userContext{
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
func (u userContext) TrackAuth(loginSuccess bool) UserContext {
	u.ctx.events.TrackUserAuth(u.id, loginSuccess)
	return u
}

// TrackAuthSuccess is equivalent to `TrackAuth(true)`.
func (u userContext) TrackAuthSuccess() UserContext {
	return u.TrackAuth(true)
}

// TrackAuthFailure is equivalent to `TrackAuth(false)`.
func (u userContext) TrackAuthFailure() UserContext {
	return u.TrackAuth(false)
}

// TrackSignup allows to track a user signup. A call to this method creates a
// new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackSignup()
//
func (u userContext) TrackSignup() UserContext {
	u.ctx.events.TrackUserSignup(u.id)
	return u
}

// TrackEvent is a convenience method to send a custom security event
// associated to the user. It is equivalent to using method
// `WithUserIdentifiers()` on the regular `TrackEvent()` method.
// So it is equivalent to
// `sdk.FromContext(ctx).TrackEvent("event").WithUserIdentifiers(uid)`.
// This alternative should be considered when performing multiple user events
// as it allows to write fewer lines.
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
func (u userContext) TrackEvent(event string) UserEvent {
	uevent := u.ctx.TrackEvent(event).WithUserIdentifiers(u.id)
	return userEvent{event: uevent}
}

type (
	// UserEvent is a custom user event. Its methods allow request handlers to
	// add options further defining the event, such as a extra properties, etc.
	UserEvent interface {
		WithProperties(properties EventPropertyMap) UserEvent
		WithTimestamp(timestamp time.Time) UserEvent
	}

	userEvent struct {
		event TrackEvent
	}
)

// WithTimestamp adds a custom timestamp to the event. By default, the timestamp
// is set to `time.Now()` value at the time of the call to the event creation.
//
//	sdk.FromContext(ctx).TrackEvent("my.event").WithTimestamp(yourTimestamp)
//
func (e userEvent) WithTimestamp(t time.Time) UserEvent {
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
func (e userEvent) WithProperties(p EventPropertyMap) UserEvent {
	e.event.WithProperties(p)
	return e
}

// Identify globally associates the given UserContext identifiers to the current
// request and returns a non-nil error if the user was blocked by Sqreen. Note
// that when an error is returned, the request was already answered with your
// blocking configuration and the request context was canceled in order to abort
// every ongoing operation. So the caller shouldn't continue handling the
// request any further.
//
// Every event following this one will be automatically associated to this
// user, unless forced using `WithUserIdentifiers()`.
//
// Usage example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	if err := sqUser.Identify(); err != nil {
//		// Return now to stop further handling the request. Returning the error
//		// may help bubbling up the handler call stack.
//		return err
//	}
//
func (u userContext) Identify() error {
	return u.ctx.events.IdentifyUser(u.id)
}

// Returned when the context value is not found.
type disabledEventRecorder struct{}

func (disabledEventRecorder) WithTimestamp(time.Time)                            {}
func (disabledEventRecorder) WithProperties(protection_context.EventProperties)  {}
func (disabledEventRecorder) WithUserIdentifiers(map[string]string)              {}
func (d disabledEventRecorder) TrackEvent(string) protection_context.CustomEvent { return d }
func (disabledEventRecorder) TrackUserSignup(map[string]string)                  {}
func (disabledEventRecorder) TrackUserAuth(map[string]string, bool)              {}
func (disabledEventRecorder) IdentifyUser(map[string]string) error               { return nil }

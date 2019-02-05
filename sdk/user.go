package sdk

import "github.com/sqreen/go-agent/agent"

// UserHTTPRequestRecord is the SDK record associated to a HTTP request for a
// given user. Its methods allow request handlers to signal security events
// related to the given user. It allows to send security events related to a
// single user.
type UserHTTPRequestRecord struct {
	ctx *agent.HTTPRequestRecord
	id  EventUserIdentifiersMap
}

// TrackAuth allows to track a user authentication. The boolean value
// `loginSuccess` must be true when the user successfully logged in, false
// otherwise. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).User(uid)
//	sqUser.TrackAuthSuccess()
//
func (ctx *UserHTTPRequestRecord) TrackAuth(loginSuccess bool) *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	ctx.ctx.TrackAuth(loginSuccess, agent.EventUserIdentifiersMap(ctx.id))
	return ctx
}

// TrackAuthSuccess is equivalent to `TrackAuth(true)`.
func (ctx *UserHTTPRequestRecord) TrackAuthSuccess() *UserHTTPRequestRecord {
	return ctx.TrackAuth(true)
}

// TrackAuthFailure is equivalent to `TrackAuth(false)`.
func (ctx *UserHTTPRequestRecord) TrackAuthFailure() *UserHTTPRequestRecord {
	return ctx.TrackAuth(false)
}

// TrackSignup allows to track a user signup. A call to this method creates a
// new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).User(uid)
//	sqUser.TrackSignup()
//
func (ctx *UserHTTPRequestRecord) TrackSignup() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	ctx.ctx.TrackSignup(agent.EventUserIdentifiersMap(ctx.id))
	return ctx
}

// TrackIdentify allows to globally identify a user for the current request.
// Every event will be automtically associated to this user. A call to this
// method does not create an event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).User(uid)
//	sqUser.TrackIdentify()
//
func (ctx *UserHTTPRequestRecord) TrackIdentify() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	ctx.ctx.TrackIdentify(agent.EventUserIdentifiersMap(ctx.id))
	return ctx
}

// TrackEvent allows to send a custom security event related to the user. A call
// to this method creates an event. Note that the top-level `TrackEvent()` does
// not associate any user unless you call the method `WithUserCredentials()`. To
// avoid confusion, the object returned does not provide `WithUserCredentials()`
// method.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).User(uid)
//	sqUser.TrackEvent("my.event")
//
func (ctx *UserHTTPRequestRecord) TrackEvent(event string) *UserHTTPRequestEvent {
	if ctx == nil {
		return nil
	}
	ctx.TrackIdentify()
	return &UserHTTPRequestEvent{&HTTPRequestEvent{ctx.ctx.TrackEvent(event)}}
}

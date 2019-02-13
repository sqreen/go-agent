package sdk

import "github.com/sqreen/go-agent/agent/types"

// UserHTTPRequestRecord is the SDK record associated to a HTTP request for a
// given user. Its methods allow request handlers to signal security events
// related to the given user. It allows to send security events related to a
// single user.
type UserHTTPRequestRecord struct {
	record types.RequestRecord
	id     EventUserIdentifiersMap
}

// TrackAuth allows to track a user authentication. The boolean value
// `loginSuccess` must be true when the user successfully logged in, false
// otherwise. A call to this method creates a new event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackAuthSuccess()
//
func (ctx *UserHTTPRequestRecord) TrackAuth(loginSuccess bool) *UserHTTPRequestRecord {
	ctx.record.NewUserAuth(ctx.id, loginSuccess)
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
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackSignup()
//
func (ctx *UserHTTPRequestRecord) TrackSignup() *UserHTTPRequestRecord {
	ctx.record.NewUserSignup(ctx.id)
	return ctx
}

// TrackEvent allows to send a custom security event related to the user. A call
// to this method creates an event. Note that this method automatically
// associates the user to the request, compared to the top-level `TrackEvent()`
// that does not, unless using its `WithUserCredentials()` method. To avoid
// confusion, the object returned does not provide `WithUserCredentials()`
// method.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.TrackEvent("my.event")
//
func (ctx *UserHTTPRequestRecord) TrackEvent(event string) *UserHTTPRequestEvent {
	ctx.record.Identify(ctx.id)
	return &UserHTTPRequestEvent{HTTPRequestEvent{ctx.record.NewCustomEvent(event)}}
}

// Identify associates the user to current request so that Sqreen can apply
// security countermeasures targeting specific users when necessary. A call to
// this method does not create an event.
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.Identify()
//
func (ctx *UserHTTPRequestRecord) Identify() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	ctx.record.Identify(ctx.id)
	return ctx
}

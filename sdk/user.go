package sdk

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/types"
)

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
	if ctx == nil {
		return nil
	}
	ctx.record.NewUserAuth(ctx.id, loginSuccess)
	return ctx
}

// TrackAuthSuccess is equivalent to `TrackAuth(true)`.
func (ctx *UserHTTPRequestRecord) TrackAuthSuccess() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	return ctx.TrackAuth(true)
}

// TrackAuthFailure is equivalent to `TrackAuth(false)`.
func (ctx *UserHTTPRequestRecord) TrackAuthFailure() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
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
	if ctx == nil {
		return nil
	}
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
	if ctx == nil {
		return nil
	}
	ctx.record.Identify(ctx.id)
	return &UserHTTPRequestEvent{HTTPRequestEvent{ctx.record.NewCustomEvent(event)}}
}

// Identify globally associates the given user-identifiers to the current
// request. A call to this method should be followed by a call to method
// `SecurityResponse()` to check if the request should be aborted.
//
// Every event happening in the same request will be therefore automatically
// associated to these user-identifiers, unless overwritten and forced using
// `WithUserIdentifiers()`.
//
// They are also required to find security responses for users, for example to
// block a specific user.
//
// This method and `MatchSecurityResponse()` are not concurrency-safe.
//
// Usage example:
//
//	uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//	sqUser := sdk.FromContext(ctx).ForUser(uid)
//	sqUser.Identify()
//	if match, err := sqUser.MatchSecurityResponse(); match {
//		// Return now to stop further handling the request and let Sqreen's
//		// middleware apply and abort the request. The returned error may help
//		// aborting from sub-functions by returning it to the callers when the
//		// Go error handling pattern is used.
//		return err
//	}
//
func (ctx *UserHTTPRequestRecord) Identify() *UserHTTPRequestRecord {
	if ctx == nil {
		return nil
	}
	ctx.record.Identify(ctx.id)
	return ctx
}

// MatchSecurityResponse returns `true` and a non-nil error if a security
// response matches the current request. The handler should stop serving the
// request by returning from the function up to Sqreen's middleware function
// which will apply the security response and abort the request.
// Note that `panic()` shouldn't be used.
//
// The returned error may help aborting from sub-functions by returning it to
// the callers when the Go error handling pattern is used.
//
// This method and `Identify()` are not concurrency-safe.
func (ctx *UserHTTPRequestRecord) MatchSecurityResponse() (match bool, err error) {
	if ctx == nil {
		return false, nil
	}

	response := ctx.record.UserSecurityResponse()
	if response != nil {
		err = SecurityResponseMatch{response}
	}
	return response != nil, err
}

// SecurityResponseMatch is an error type wrapping the security response that
// matched the request and helping in bubbling up to Sqreen's middleware
// function to abort the request.
type SecurityResponseMatch struct {
	Handler http.Handler
}

func (SecurityResponseMatch) Error() string {
	return "a security response matched the request"
}

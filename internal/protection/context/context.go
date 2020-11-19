// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package context

import (
	"encoding/json"
	"time"
)

// ContextKey is the context key value to use in order to store a protection
// context into a request context in HTTP middlewares before calling the
// handler.
var ContextKey = contextKey{"_sqreen_ctx_"}

// ContextKey allows to insert context values avoiding string collisions. Cf.
// `context.WithValue()`.
type contextKey struct {
	// This string value must be used by middleware functions whose framework
	// expects context keys of type string, such as Gin. `sdk.FromContext()`
	// expect this behaviour to fallback to string keys when getting the value
	// from the pointer address returned null.
	String string
}

type (
	// EventRecorder is the interface used by the SDK.
	EventRecorder interface {
		// TrackEvent records a new custom event and returns its object to
		// further specify it if required.
		TrackEvent(event string) CustomEvent
		// TrackUserSignup records a new user signup event.
		TrackUserSignup(id map[string]string)
		// TrackUserAuth records a new user authentication (sign-in) event.
		TrackUserAuth(id map[string]string, success bool)
		// IdentifyUser globally associates the given user identifiers to the current
		// request. An non-nil error is returned when a security response matches
		// the given user id.
		IdentifyUser(id map[string]string) error
	}

	CustomEvent interface {
		WithTimestamp(t time.Time)
		WithProperties(props EventProperties)
		WithUserIdentifiers(id map[string]string)
	}

	// EventProperties is an interface type enforcing a marshalable type to the
	// JSON format.
	EventProperties json.Marshaler
)

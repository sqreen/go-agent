// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Security Actions
//
// Security actions are stored using structures implementing the Action
// interface. Actions can have a time duration by implementing the Timed
// interface.
//
package actor

import (
	"time"
)

// Action kinds.
const (
	actionKind_BlockIP   = "block_ip"
	actionKind_BlockUser = "block_user"
)

// Action is an interface common to each concrete action type stored in the data
// structures, and allowing to type-switch the stored values.
type Action interface {
	// ActionID returns the unique ID of the request.
	ActionID() string
}

// Timed is an interface implemented by actions having an expiration time.
type Timed interface {
	Expired() bool
}

type blockAction struct {
	ID string
}

func newBlockAction(id string) *blockAction {
	return &blockAction{
		ID: id,
	}
}

func (a *blockAction) ActionID() string {
	return a.ID
}

// timedAction is an Action with a time deadline after which it is considered
// expired.
type timedAction struct {
	Action
	deadline time.Time
}

// withDuration sets a time duration to an action. The returned value implements
// the Action and Timed interfaces.
func withDuration(action Action, duration time.Duration) *timedAction {
	return &timedAction{
		Action:   action,
		deadline: time.Now().Add(duration),
	}
}

// Expired is true when the deadline has expired, false otherwise.
func (a *timedAction) Expired() bool {
	// Is the current time after the deadline?
	return time.Now().After(a.deadline)
}

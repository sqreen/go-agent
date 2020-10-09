// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package event

import (
	"net"
	"sync"
	"time"

	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
)

type Record struct {
	customEventsLock sync.Mutex
	customEvents     []*CustomEvent

	userEventsLock sync.Mutex
	userEvents     []UserEventFace

	attacksLock sync.Mutex
	attacks     []*AttackEvent

	// A user id has been globally associated using `Identify()`
	identifiedUser bool
}

type PropertyMap = map[string]string

type UserIdentifierMap = map[string]string

type CustomEvent struct {
	Method     string
	Event      string
	Properties protectioncontext.EventProperties
	UserID     UserIdentifierMap
	Timestamp  time.Time
}

type AttackEvent struct {
	Rule       string
	Test       bool
	Blocked    bool
	Timestamp  time.Time
	Info       interface{}
	StackTrace []uintptr
	AttackType string
}

func (r *Record) AddAttackEvent(attack interface{}) {
	actual, ok := attack.(*AttackEvent)
	if !ok {
		return
	}

	r.attacksLock.Lock()
	defer r.attacksLock.Unlock()
	r.attacks = append(r.attacks, actual)
}

func (r *Record) AddUserAuth(id UserIdentifierMap, ip net.IP, success bool) {
	if len(id) == 0 {
		return
	}
	event := &AuthUserEvent{
		LoginSuccess: success,
		UserEvent: &UserEvent{
			UserIdentifiers: id,
			IP:              ip,
			Timestamp:       time.Now(),
		},
	}
	r.addUserEvent(event)
}

func (r *Record) AddUserSignup(id UserIdentifierMap, ip net.IP) {
	if len(id) == 0 {
		return
	}

	event := &SignupUserEvent{
		UserEvent: &UserEvent{
			UserIdentifiers: id,
			IP:              ip,
			Timestamp:       time.Now(),
		},
	}
	r.addUserEvent(event)
}

// TODO: rename as "metrics" or "observations"
type UserEventFace interface {
	isUserEvent()
}

type UserEvent struct {
	UserIdentifiers UserIdentifierMap
	Timestamp       time.Time
	IP              net.IP
}

type AuthUserEvent struct {
	*UserEvent
	LoginSuccess bool
}

func (*AuthUserEvent) isUserEvent() {}

type SignupUserEvent struct {
	*UserEvent
}

func (*SignupUserEvent) isUserEvent() {}

const (
	SDKMethodIdentify = "identify"
	SDKMethodTrack    = "track"
)

func (r *Record) AddCustomEvent(name string) *CustomEvent {
	event := &CustomEvent{
		Method:    SDKMethodTrack,
		Event:     name,
		Timestamp: time.Now(),
	}
	r.addEvent(event)
	return event
}

// Globally associate these user-identifiers with the request.
func (r *Record) Identify(id UserIdentifierMap) {
	if r.identifiedUser {
		return
	}
	r.identifiedUser = true
	evt := &CustomEvent{
		Method:    SDKMethodIdentify,
		UserID:    id,
		Timestamp: time.Now(),
	}
	r.addEvent(evt)
}

func (r *Record) addEvent(event *CustomEvent) {
	r.customEventsLock.Lock()
	defer r.customEventsLock.Unlock()
	r.customEvents = append(r.customEvents, event)
}

func (r *Record) addUserEvent(event UserEventFace) {
	// User customEvents don't go through the request record Event list but through
	// aggregated metrics.
	r.userEventsLock.Lock()
	defer r.userEventsLock.Unlock()
	r.userEvents = append(r.userEvents, event)
}

func (e *CustomEvent) WithTimestamp(t time.Time) {
	e.Timestamp = t
}

func (e *CustomEvent) WithProperties(p protectioncontext.EventProperties) {
	e.Properties = p
}

func (e *CustomEvent) WithUserIdentifiers(id UserIdentifierMap) {
	e.UserID = id
}

type Recorded struct {
	AttackEvents []*AttackEvent
	CustomEvents []*CustomEvent
	UserEvents   []UserEventFace
}

func (r *Record) CloseRecord() Recorded {
	return Recorded{
		AttackEvents: r.flushAttackEvents(),
		CustomEvents: r.flushCustomEvents(),
		UserEvents:   r.flushUserEvents(),
	}
}

func (r *Record) flushUserEvents() []UserEventFace {
	r.userEventsLock.Lock()
	defer r.userEventsLock.Unlock()
	events := r.userEvents
	r.userEvents = nil
	return events
}

func (r *Record) flushAttackEvents() []*AttackEvent {
	r.attacksLock.Lock()
	defer r.attacksLock.Unlock()
	events := r.attacks
	r.attacks = nil
	return events
}

func (r *Record) flushCustomEvents() []*CustomEvent {
	r.customEventsLock.Lock()
	defer r.customEventsLock.Unlock()
	events := r.customEvents
	r.customEvents = nil
	return events
}

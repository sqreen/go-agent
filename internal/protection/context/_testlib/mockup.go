// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package _testlib

import (
	"time"

	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/stretchr/testify/mock"
)

type EventRecorderMockup struct {
	mock.Mock
}

var (
	_ protection_context.EventRecorder = (*EventRecorderMockup)(nil)
	_ protection_context.CustomEvent   = (*EventRecorderMockup)(nil)
)

func (e *EventRecorderMockup) TrackEvent(event string) protection_context.CustomEvent {
	r, _ := e.Called(event).Get(0).(protection_context.CustomEvent)
	return r
}

func (e *EventRecorderMockup) ExpectTrackEvent(event string) *mock.Call {
	return e.On("TrackEvent", event)
}

func (e *EventRecorderMockup) TrackUserSignup(id map[string]string) {
	e.Called(id)
}

func (e *EventRecorderMockup) ExpectTrackUserSignup(id map[string]string) *mock.Call {
	return e.On("TrackUserSignup", id)
}

func (e *EventRecorderMockup) TrackUserAuth(id map[string]string, success bool) {
	e.Called(id, success)
}

func (e *EventRecorderMockup) ExpectTrackUserAuth(id map[string]string, success bool) *mock.Call {
	return e.On("TrackUserAuth", id, success)
}

func (e *EventRecorderMockup) IdentifyUser(id map[string]string) error {
	return e.Called(id).Error(0)
}

func (e *EventRecorderMockup) ExpectIdentifyUser(id map[string]string) *mock.Call {
	return e.On("IdentifyUser", id)
}

func (e *EventRecorderMockup) WithTimestamp(t time.Time) {
	e.Called(t)
}

func (e *EventRecorderMockup) ExpectWithTimestamp(t time.Time) *mock.Call {
	return e.On("WithTimestamp", t)
}

func (e *EventRecorderMockup) WithProperties(props protection_context.EventProperties) {
	e.Called(props)
}

func (e *EventRecorderMockup) ExpectWithProperties(props protection_context.EventProperties) *mock.Call {
	return e.On("WithProperties", props)
}

func (e *EventRecorderMockup) WithUserIdentifiers(id map[string]string) {
	e.Called(id)
}

func (e *EventRecorderMockup) ExpectWithUserIdentifiers(id map[string]string) *mock.Call {
	return e.On("WithUserIdentifiers", id)
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/context/_testlib"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFromContext(t *testing.T) {
	t.Run("unset value", func(t *testing.T) {
		sq := sdk.FromContext(context.Background())
		dummySDKTest(t, sq)
	})

	t.Run("nil context", func(t *testing.T) {
		sq := sdk.FromContext(nil)
		dummySDKTest(t, sq)
	})

	t.Run("from a pointer key", func(t *testing.T) {
		recorder := &_testlib.EventRecorderMockup{}
		recorder.ExpectTrackEvent(mock.Anything).Return(recorder)
		recorder.ExpectIdentifyUser(mock.Anything).Return(nil)
		recorder.ExpectTrackUserAuth(mock.Anything, mock.AnythingOfType("bool")).Return(recorder)
		recorder.ExpectTrackUserSignup(mock.Anything).Return(recorder)
		recorder.ExpectWithUserIdentifiers(mock.Anything).Return(recorder)
		recorder.ExpectWithTimestamp(mock.Anything).Return(recorder)
		recorder.ExpectWithProperties(mock.Anything).Return(recorder)

		ctx := context.WithValue(context.Background(), protection_context.ContextKey, recorder)
		sq := sdk.FromContext(ctx)
		dummySDKTest(t, sq)
	})

	t.Run("from a string key", func(t *testing.T) {
		recorder := &_testlib.EventRecorderMockup{}
		recorder.ExpectTrackEvent(mock.Anything).Return(recorder)
		recorder.ExpectIdentifyUser(mock.Anything).Return(nil)
		recorder.ExpectTrackUserAuth(mock.Anything, mock.AnythingOfType("bool")).Return(recorder)
		recorder.ExpectTrackUserSignup(mock.Anything).Return(recorder)
		recorder.ExpectWithUserIdentifiers(mock.Anything).Return(recorder)
		recorder.ExpectWithTimestamp(mock.Anything).Return(recorder)
		recorder.ExpectWithProperties(mock.Anything).Return(recorder)

		ctx := context.WithValue(context.Background(), protection_context.ContextKey.String, recorder)
		sq := sdk.FromContext(ctx)
		dummySDKTest(t, sq)
	})

	t.Run("wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), protection_context.ContextKey, 33)
		sq := sdk.FromContext(ctx)
		dummySDKTest(t, sq)
	})
}

func TestTrackEvent(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		ctx, recorder := newMockups()
		defer recorder.AssertExpectations(t)

		trackEventRecorder := &_testlib.EventRecorderMockup{}
		defer recorder.AssertExpectations(t)

		eventID := testlib.RandUTF8String(1, 50)
		recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

		sdk.FromContext(ctx).TrackEvent(eventID)
	})

	t.Run("with user identifiers", func(t *testing.T) {
		ctx, recorder := newMockups()
		defer recorder.AssertExpectations(t)

		trackEventRecorder := &_testlib.EventRecorderMockup{}
		defer trackEventRecorder.AssertExpectations(t)

		eventID := testlib.RandUTF8String(1, 50)
		recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

		userID := sdk.EventUserIdentifiersMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
		trackEventRecorder.ExpectWithUserIdentifiers(map[string]string(userID)).Once()

		sdk.FromContext(ctx).TrackEvent(eventID).WithUserIdentifiers(userID)
	})

	t.Run("with properties", func(t *testing.T) {
		ctx, recorder := newMockups()
		defer recorder.AssertExpectations(t)

		trackEventRecorder := &_testlib.EventRecorderMockup{}
		defer trackEventRecorder.AssertExpectations(t)

		eventID := testlib.RandUTF8String(1, 50)
		recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

		properties := sdk.EventPropertyMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
		trackEventRecorder.ExpectWithProperties(properties).Once()

		sdk.FromContext(ctx).TrackEvent(eventID).WithProperties(properties)
	})

	t.Run("with timestamp", func(t *testing.T) {
		ctx, recorder := newMockups()
		defer recorder.AssertExpectations(t)

		trackEventRecorder := &_testlib.EventRecorderMockup{}
		defer trackEventRecorder.AssertExpectations(t)

		eventID := testlib.RandUTF8String(1, 50)
		recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

		ts := time.Now()
		trackEventRecorder.ExpectWithTimestamp(ts).Once()

		sdk.FromContext(ctx).TrackEvent(eventID).WithTimestamp(ts)
	})

	t.Run("multiple options chaining", func(t *testing.T) {
		testWithTimestamp := func(t *testing.T, event sdk.TrackEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.TrackEvent {
			ts := time.Now()
			trackEventRecorder.ExpectWithTimestamp(ts).Once()
			return event.WithTimestamp(ts)
		}

		testWithProperties := func(t *testing.T, event sdk.TrackEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.TrackEvent {
			properties := sdk.EventPropertyMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
			trackEventRecorder.ExpectWithProperties(properties).Once()
			return event.WithProperties(properties)
		}

		testWithUserIdentifiers := func(t *testing.T, event sdk.TrackEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.TrackEvent {
			uid := sdk.EventUserIdentifiersMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
			trackEventRecorder.ExpectWithUserIdentifiers(map[string]string(uid)).Once()
			return event.WithUserIdentifiers(uid)
		}

		tests := []func(t *testing.T, event sdk.TrackEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.TrackEvent{
			testWithTimestamp,
			testWithProperties,
			testWithUserIdentifiers,
		}
		for _, i := range tests {
			for _, j := range tests {
				for _, k := range tests {
					t.Run("", func(t *testing.T) {
						ctx, recorder := newMockups()
						defer recorder.AssertExpectations(t)

						trackEventRecorder := &_testlib.EventRecorderMockup{}
						defer trackEventRecorder.AssertExpectations(t)

						eventID := testlib.RandUTF8String(1, 50)
						recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

						event := sdk.FromContext(ctx).TrackEvent(eventID)

						event = i(t, event, trackEventRecorder)
						event = j(t, event, trackEventRecorder)
						event = k(t, event, trackEventRecorder)
					})
				}
			}
		}
	})
}

func TestForUser(t *testing.T) {
	prepareUserContext := func() (uid sdk.EventUserIdentifiersMap, event sdk.UserContext, recorder *_testlib.EventRecorderMockup) {
		ctx, recorder := newMockups()
		uid = sdk.EventUserIdentifiersMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
		return uid, sdk.FromContext(ctx).ForUser(uid), recorder
	}

	t.Run("TrackEvent", func(t *testing.T) {
		prepareTrackEvent := func() (event sdk.UserEvent, recorder, trackEventRecorder *_testlib.EventRecorderMockup) {
			uid, sqUser, recorder := prepareUserContext()

			trackEventRecorder = &_testlib.EventRecorderMockup{}

			eventID := testlib.RandUTF8String(1, 50)
			recorder.ExpectTrackEvent(eventID).Return(trackEventRecorder).Once()

			trackEventRecorder.ExpectWithUserIdentifiers(map[string]string(uid))

			return sqUser.TrackEvent(eventID), recorder, trackEventRecorder
		}

		t.Run("without options", func(t *testing.T) {
			_, recorder, trackEventRecorder := prepareTrackEvent()
			defer recorder.AssertExpectations(t)
			defer trackEventRecorder.AssertExpectations(t)
		})

		t.Run("with properties", func(t *testing.T) {
			event, recorder, trackEventRecorder := prepareTrackEvent()
			defer recorder.AssertExpectations(t)
			defer trackEventRecorder.AssertExpectations(t)

			properties := sdk.EventPropertyMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
			trackEventRecorder.ExpectWithProperties(properties).Once()

			event.WithProperties(properties)
		})

		t.Run("with timestamp", func(t *testing.T) {
			event, recorder, trackEventRecorder := prepareTrackEvent()
			defer recorder.AssertExpectations(t)
			defer trackEventRecorder.AssertExpectations(t)

			ts := time.Now()
			trackEventRecorder.ExpectWithTimestamp(ts).Once()

			event.WithTimestamp(ts)
		})

		t.Run("multiple options chaining", func(t *testing.T) {
			testWithTimestamp := func(t *testing.T, event sdk.UserEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.UserEvent {
				ts := time.Now()
				trackEventRecorder.ExpectWithTimestamp(ts).Once()
				return event.WithTimestamp(ts)
			}

			testWithProperties := func(t *testing.T, event sdk.UserEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.UserEvent {
				properties := sdk.EventPropertyMap{testlib.RandUTF8String(1, 50): testlib.RandUTF8String(1, 50)}
				trackEventRecorder.ExpectWithProperties(properties).Once()
				return event.WithProperties(properties)
			}

			tests := []func(t *testing.T, event sdk.UserEvent, trackEventRecorder *_testlib.EventRecorderMockup) sdk.UserEvent{
				testWithTimestamp,
				testWithProperties,
			}
			for _, i := range tests {
				for _, j := range tests {
					for _, k := range tests {
						t.Run("", func(t *testing.T) {
							event, recorder, trackEventRecorder := prepareTrackEvent()
							defer recorder.AssertExpectations(t)
							defer trackEventRecorder.AssertExpectations(t)

							event = i(t, event, trackEventRecorder)
							event = j(t, event, trackEventRecorder)
							event = k(t, event, trackEventRecorder)
						})
					}
				}
			}
		})
	})

	t.Run("TrackSignup", func(t *testing.T) {
		uid, sqUser, recorder := prepareUserContext()
		defer recorder.AssertExpectations(t)
		recorder.ExpectTrackUserSignup(map[string]string(uid))
		sqUser.TrackSignup()
	})

	t.Run("TrackAuth", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			uid, sqUser, recorder := prepareUserContext()
			defer recorder.AssertExpectations(t)
			recorder.ExpectTrackUserAuth(map[string]string(uid), true)
			sqUser.TrackAuth(true)
		})

		t.Run("false", func(t *testing.T) {
			uid, sqUser, recorder := prepareUserContext()
			defer recorder.AssertExpectations(t)
			recorder.ExpectTrackUserAuth(map[string]string(uid), false)
			sqUser.TrackAuth(false)
		})
	})

	t.Run("TrackAuthFailure", func(t *testing.T) {
		uid, sqUser, recorder := prepareUserContext()
		defer recorder.AssertExpectations(t)
		recorder.ExpectTrackUserAuth(map[string]string(uid), false)
		sqUser.TrackAuthFailure()
	})

	t.Run("TrackAuthSuccess", func(t *testing.T) {
		uid, sqUser, recorder := prepareUserContext()
		defer recorder.AssertExpectations(t)
		recorder.ExpectTrackUserAuth(map[string]string(uid), true)
		sqUser.TrackAuthSuccess()
	})

	t.Run("Identify", func(t *testing.T) {
		t.Run("no error returned", func(t *testing.T) {
			uid, sqUser, recorder := prepareUserContext()
			defer recorder.AssertExpectations(t)
			recorder.ExpectIdentifyUser(map[string]string(uid)).Return(nil).Once()
			require.NoError(t, sqUser.Identify())
		})

		t.Run("error returned", func(t *testing.T) {
			uid, sqUser, recorder := prepareUserContext()
			defer recorder.AssertExpectations(t)
			userErr := errors.New("blocked user")
			recorder.ExpectIdentifyUser(map[string]string(uid)).Return(userErr).Once()
			require.Equal(t, userErr, sqUser.Identify())
		})
	})
}

func TestEventPropertyMap(t *testing.T) {
	key := testlib.RandPrintableUSASCIIString(1, 100)
	value := testlib.RandPrintableUSASCIIString(1, 100)
	props := sdk.EventPropertyMap{
		key: value,
	}
	buf, err := props.MarshalJSON()
	require.NoError(t, err)
	expected, err := json.Marshal(map[string]string{key: value})
	require.NoError(t, err)
	require.Equal(t, string(expected), string(buf))
}

func dummySDKTest(t *testing.T, sqreen sdk.Context) {
	event := sqreen.TrackEvent(testlib.RandPrintableUSASCIIString(0, 50))
	event = event.WithTimestamp(time.Now())
	userID := sdk.EventUserIdentifiersMap{testlib.RandPrintableUSASCIIString(2, 30): testlib.RandPrintableUSASCIIString(2, 30)}
	event = event.WithUserIdentifiers(userID)
	props := sdk.EventPropertyMap{testlib.RandPrintableUSASCIIString(2, 30): testlib.RandPrintableUSASCIIString(2, 30)}
	event = event.WithProperties(props)
	uid := sdk.EventUserIdentifiersMap{testlib.RandPrintableUSASCIIString(2, 30): testlib.RandPrintableUSASCIIString(2, 30)}
	sqUser := sqreen.ForUser(uid)
	sqUser = sqUser.TrackSignup()
	sqUser = sqUser.TrackAuth(true)
	sqUser = sqUser.TrackAuthSuccess()
	sqUser = sqUser.TrackAuthFailure()
	require.NoError(t, sqUser.Identify())
	sqUserEvent := sqUser.TrackEvent(testlib.RandPrintableUSASCIIString(0, 50))
	sqUserEvent = sqUserEvent.WithProperties(props)
	sqUserEvent = sqUserEvent.WithTimestamp(time.Now())
}

func newMockups() (context.Context, *_testlib.EventRecorderMockup) {
	recorder := &_testlib.EventRecorderMockup{}
	ctx := context.WithValue(context.Background(), protection_context.ContextKey, recorder)
	return ctx, recorder
}

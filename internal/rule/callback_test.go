// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"errors"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/rule/callback/_testlib/mockups"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
	"github.com/sqreen/go-agent/tools/testlib/testmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func TestCallbackMiddlewares(t *testing.T) {
	t.Run("withSafeCall", func(t *testing.T) {
		t.Run("no panic", func(t *testing.T) {
			logger := &testmock.LoggerMockup{}
			// No error should be logged
			defer logger.AssertExpectations(t)

			m := withSafeCall()
			var called bool
			cb := m(func(c callback.CallbackContext) error {
				called = true
				return nil
			})
			require.NoError(t, cb(nil))
			require.True(t, called)
		})

		t.Run("panic", func(t *testing.T) {
			m := withSafeCall()
			cb := m(func(c callback.CallbackContext) error {
				panic("oops")
				return nil
			})
			err := cb(nil)
			require.Error(t, err)

			expected := &sqsafe.PanicError{}
			require.True(t, xerrors.As(err, &expected))
			require.Equal(t, expected.Err.Error(), "oops")
		})

		t.Run("error", func(t *testing.T) {
			m := withSafeCall()
			errOops := errors.New("oops")
			cb := m(func(c callback.CallbackContext) error {
				return errOops
			})
			err := cb(nil)
			require.Error(t, err)

			require.Equal(t, errOops, err)
		})

	})

	t.Run("withCallCount", func(t *testing.T) {
		timeHist := &testmock.TimeHistogramMockup{}
		defer timeHist.AssertExpectations(t)

		// We expect the call counter to be increased three times
		expectedKey := "pack/rule/callback"
		timeHist.ExpectAdd(expectedKey, 1).Return(nil).Times(3)

		m := withCallCount("pack", "rule", "callback", timeHist)

		var called int
		h := m(func(c callback.CallbackContext) error {
			called += 1
			return nil
		})

		// Call the handler 3 times
		require.NoError(t, h(nil))
		require.NoError(t, h(nil))
		require.NoError(t, h(nil))

		require.Equal(t, 3, called)
	})

	t.Run("withPerformanceCap", func(t *testing.T) {
		t.Run("deadline not exceeded", func(t *testing.T) {
			c := &mockups.CallbackContextMockup{}
			defer c.AssertExpectations(t)

			p := &mockups.ProtectionContextMockup{}
			defer p.AssertExpectations(t)

			c.ExpectProtectionContext().Return(p)
			// Deadline not exceeded
			p.ExpectDeadlineExceeded(0).Return(false).Twice()

			timeHist := &testmock.TimeHistogramMockup{}
			defer timeHist.AssertExpectations(t)

			m := withPerformanceCap("rule", timeHist)
			var called bool
			cb := m(func(c callback.CallbackContext) error {
				called = true
				return nil
			})

			require.NoError(t, cb(c))
			require.True(t, called)
		})

		t.Run("deadline exceeded", func(t *testing.T) {
			t.Run("before the callback", func(t *testing.T) {
				c := &mockups.CallbackContextMockup{}
				defer c.AssertExpectations(t)

				p := &mockups.ProtectionContextMockup{}
				defer p.AssertExpectations(t)

				c.ExpectProtectionContext().Return(p)
				// Deadline exceeded
				p.ExpectDeadlineExceeded(0).Return(true).Once()

				timeHist := &testmock.TimeHistogramMockup{}
				timeHist.ExpectAdd("rule/before", 1).Return(nil).Once()
				defer timeHist.AssertExpectations(t)

				m := withPerformanceCap("rule", timeHist)
				var called bool
				cb := m(func(c callback.CallbackContext) error {
					called = true
					return nil
				})

				require.NoError(t, cb(c))
				require.False(t, called)
			})

			t.Run("after the callback", func(t *testing.T) {
				c := &mockups.CallbackContextMockup{}
				defer c.AssertExpectations(t)

				p := &mockups.ProtectionContextMockup{}
				defer p.AssertExpectations(t)

				c.ExpectProtectionContext().Return(p)
				// Deadline not exceeded before the callback
				p.ExpectDeadlineExceeded(0).Return(false).Once()
				// Deadline exceeded after the callback
				p.ExpectDeadlineExceeded(0).Return(true).Once()

				timeHist := &testmock.TimeHistogramMockup{}
				timeHist.ExpectAdd("rule/after", 1).Return(nil).Once()
				defer timeHist.AssertExpectations(t)

				m := withPerformanceCap("rule", timeHist)
				var called bool
				cb := m(func(c callback.CallbackContext) error {
					called = true
					return nil
				})

				require.NoError(t, cb(c))
				require.True(t, called)
			})
		})
	})

	t.Run("withPerformanceMonitoring", func(t *testing.T) {
		c := &mockups.CallbackContextMockup{}
		defer c.AssertExpectations(t)

		p := &mockups.ProtectionContextMockup{}
		defer p.AssertExpectations(t)

		c.ExpectProtectionContext().Return(p)

		sqreenTime := sqtime.NewSharedStopWatch()
		p.ExpectSqreenTime().Return(sqreenTime).Once()

		perfHist := &testmock.PerfHistogramMockup{}
		defer perfHist.AssertExpectations(t)

		sleep := time.Millisecond
		var perf float64
		perfHist.ExpectAdd(mock.MatchedBy(func(v float64) bool {
			perf = v
			return v >= 1.0 // v is in ms - so >= sleep is equivalent to >= 1.0
		})).Return(nil).Once()

		m := withPerformanceMonitoring(perfHist)
		var called bool
		cb := m(func(c callback.CallbackContext) error {
			called = true
			time.Sleep(sleep)
			return nil
		})

		require.NoError(t, cb(c))

		require.True(t, called)
		require.Equal(t, perf, float64(sqreenTime.Duration().Nanoseconds())/float64(time.Millisecond))
	})
}

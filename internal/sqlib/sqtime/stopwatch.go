// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqtime

import (
	"sync"
	"sync/atomic"
	"time"
)

type (
	SharedStopWatch struct {
		lock        sync.RWMutex
		ongoing     int32
		oldestStart time.Time
		duration    time.Duration
	}

	LocalStopWatch struct {
		s                   *SharedStopWatch
		durationWhenStarted time.Duration
	}
)

func (s *SharedStopWatch) Start() (ls LocalStopWatch) {
	ls = LocalStopWatch{s: s}

	if s.updateOngoingCount(1) > 1 {
		// Hot path: the stopwatch is started
		// Set the duration since the first start time
		s.lock.RLock()
		ls.durationWhenStarted = time.Since(s.oldestStart)
		s.lock.RUnlock()
		return
	}

	// Slow path: the stopwatch is not started and we need to exclusively lock it
	// to save the current time.
	s.lock.Lock()
	defer s.lock.Unlock()

	// Exclusive lock acquired: check if we really are the first.
	if s.oldestStart.IsZero() {
		// We are the first one getting the lock and so we can set the start time
		s.oldestStart = time.Now()
		return ls // durationWhenStarted is 0
	}

	// We are not the first one getting the lock
	ls.durationWhenStarted = time.Since(s.oldestStart)
	return ls
}

func (ls *LocalStopWatch) Stop() time.Duration {
	s := ls.s

	if s.updateOngoingCount(-1) >= 1 {
		// Hot path: the stopwatch is still ongoing
		// Return the duration of the  since the first start time
		s.lock.RLock()
		dt := time.Since(s.oldestStart) - ls.durationWhenStarted
		s.lock.RUnlock()
		return dt
	}

	// The ongoing counter is back to 0 - every stopwatch was stopped so we can
	// reset the first oldest start time to zero and update the overall stopwatch
	// duration
	s.lock.Lock()
	oldestStart := s.oldestStart
	s.oldestStart = time.Time{}
	s.lock.Unlock()

	// Update the overall stopwatch duration
	dt := time.Since(oldestStart)
	atomic.AddInt64((*int64)(&s.duration), int64(dt))

	return dt - ls.durationWhenStarted
}

func (s *SharedStopWatch) updateOngoingCount(delta int32) (ongoing int32) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return atomic.AddInt32(&s.ongoing, delta)
}

func (s *SharedStopWatch) Duration() time.Duration {
	return (time.Duration)(atomic.LoadInt64((*int64)(&s.duration)))
}

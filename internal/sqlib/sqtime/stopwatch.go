// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqtime

import (
	"sync"
	"sync/atomic"
	"time"
)

type SharedStopWatch struct {
	lock        sync.RWMutex
	ongoing     int32
	oldestStart time.Time
	duration    time.Duration
}

func (s *SharedStopWatch) Start() {
	// Hot path: the stopwatch is started
	if s.fastStart(1) > 1 {
		return
	}

	// Slow path: the stopwatch is not started and we need to exclusively lock it
	// to save the current time.
	s.lock.Lock()
	defer s.lock.Unlock()
	// Exclusive lock acquired: check if we really are the first.
	if s.oldestStart.IsZero() {
		s.oldestStart = time.Now()
	}
}

func (s *SharedStopWatch) fastStart(delta int32) int32 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return atomic.AddInt32(&s.ongoing, delta)
}

func (s *SharedStopWatch) Stop() {
	if s.fastStart(-1) != 0 {
		return
	}

	s.lock.Lock()
	start := s.oldestStart
	s.oldestStart = time.Time{}
	s.lock.Unlock()

	delta := int64(time.Since(start))
	atomic.AddInt64((*int64)(&s.duration), int64(delta))
}

func (s *SharedStopWatch) Duration() time.Duration {
	return (time.Duration)(atomic.LoadInt64((*int64)(&s.duration)))
}

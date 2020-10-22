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
		zeroT       time.Time
		ongoing     int32
		oldestStart time.Duration
		duration    time.Duration
	}

	LocalStopWatch struct {
		s           *SharedStopWatch
		start, stop time.Duration
	}
)

func NewSharedStopWatch() *SharedStopWatch {
	return &SharedStopWatch{
		zeroT:       time.Now(),
		oldestStart: -1,
	}
}

func (s *SharedStopWatch) Start() (ls LocalStopWatch) {
	ls = LocalStopWatch{
		s:     s,
		start: time.Since(s.zeroT),
		stop:  -1,
	}

	if atomic.AddInt32(&s.ongoing, 1) > 1 {
		return
	}

	// Exclusively lock the watch to enforce the coherency of oldestStart's value
	// with method Stop()
	s.lock.Lock()
	defer s.lock.Unlock()

	// Check if we really are the first one
	if s.oldestStart == -1 {
		// Set oldestStart to now
		s.oldestStart = ls.start
	}

	return
}

func (ls *LocalStopWatch) Stop() (dt time.Duration) {
	if ls.stop != -1 {
		// Already stopped
		return ls.stop - ls.start
	}

	s := ls.s
	ls.stop = time.Since(s.zeroT)

	if atomic.AddInt32(&s.ongoing, -1) == 0 {
		// Exclusively lock the watch to enforce the coherency of oldestStart's
		// value with method Start()
		s.lock.Lock()
		defer s.lock.Unlock()

		// Check if we really are the last one
		if s.oldestStart != -1 {
			// Update the global duration
			duration := atomic.LoadInt64((*int64)(&s.duration))
			atomic.StoreInt64((*int64)(&s.duration), duration+int64(ls.stop-s.oldestStart))
			// Reset oldestStart
			s.oldestStart = -1
		}
	}

	return ls.stop - ls.start
}

func (s *SharedStopWatch) Duration() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&s.duration)))
}

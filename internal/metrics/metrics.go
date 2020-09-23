// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

// Package metrics provides shared metrics stores. A metrics store is a
// key/value store with a given time period after which the data is considered
// ready. This package provides an implementation optimized for writes updating
// already existing keys: lots of goroutines updating a smaller set of keys.
// The metrics engine allows to create and register new metrics stores that a
// single reader (Sqreen's agent) can concurrently read. Read and write
// operations and mutually exclusive - slow polling is better for aggregating
// more data while not blocking the writers too often.
//
// Main requirements:
//
// - Loss-less kv-stores.
// - Near zero time impact on the hot path (updates): no need to switch to
//   another goroutines, no blocking locks.
//
// Design decisions:
//
// The former first implementation was using channels and dedicated goroutines
// sleeping until the period was passed. The major issue was the case when
// the channels were full, with the choice of either blocking the sending
// goroutine, or dropping the data to avoid blocking it.
// This design is now considered not suitable for metrics as they happen at a
// too frequently to go through a channel. A channel indeed needs at least one
// extra reader goroutine that would require too much CPU time to aggregate
// all the metrics values.
//
// Metrics store operations, insertions and updates of integer values, are
// therefore considered shorter than any "pure-Go" approach with channels and
// so on. The main challenge here comes from the map whose index cannot be
// modified concurrently. So the idea is to use a RWLock it in order to
// mutually exclude the insertions of new values, updates of existing values and
// retrieval of expired values.
// The hot path being updates of existing values, the Add() method first tries
// to only RLock the store in order to avoid locking every other
// updating-goroutine. The value being a uint64, it can be atomically updated
// without using an lock for the value.
//
// The metrics stores and engine provide a polling interface to retrieve stores
// whose period are passed. No goroutine is started to automatically swap the
// stores. This is due to the fact that metrics are sent by the Sqreen agent
// only during the heartbeat; it can therefore check for expired stores.
// Metrics stores can therefore be longer than their period and will actually
// last until they are flushed by the reader goroutine.
package metrics

import (
	"fmt"
	"sync"
	"time"
)

// Engine manages the metrics stores in oder to create new one, and to poll
// the existing ones. Engine's methods are not thread-safe and designed to be
// used by a single goroutine.
type Engine struct {
	stores map[string]Store
	lock   sync.RWMutex
}

type Store interface {
	Ready() bool
	Flush() []ReadyStore
}

type ReadyStore interface {
	Start() time.Time
	Finish() time.Time
	Metrics() ReadyStoreMap
}

func NewEngine() *Engine {
	return &Engine{
		stores: make(map[string]Store),
	}
}

// TimeHistogram creates and registers a new metrics store when it does not exist. It
// returns the existing one otherwise.
func (e *Engine) TimeHistogram(id string, period time.Duration) *TimeHistogram {
	if store := e.getTimeHistogram(id); store != nil {
		return store
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if store := e.getTimeHistogramUnsafe(id); store != nil {
		return store
	}

	store := NewTimeHistogram(period, 0)
	e.stores[id] = store
	return store
}

func (e *Engine) getTimeHistogram(id string) *TimeHistogram {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.getTimeHistogramUnsafe(id)
}

func (e *Engine) getTimeHistogramUnsafe(id string) *TimeHistogram {
	if store, exists := e.stores[id]; exists {
		return store.(*TimeHistogram)
	}
	return nil
}

func (e *Engine) PerfHistogram(id string, unit, base float64, period time.Duration) (*PerfHistogram, error) {
	if store := e.getPerfHistogram(id); store != nil {
		return store, nil
	}

	e.lock.Lock()
	defer e.lock.Unlock()

	if store := e.getPerfHistogramUnsafe(id); store != nil {
		return store, nil
	}

	store, err := NewPerfHistogram(period, unit, base, 0)
	if err != nil {
		return nil, err
	}
	e.stores[id] = store
	return store, nil
}

func (e *Engine) getPerfHistogram(id string) *PerfHistogram {
	e.lock.RLock()
	defer e.lock.RUnlock()
	return e.getPerfHistogramUnsafe(id)
}

func (e *Engine) getPerfHistogramUnsafe(id string) *PerfHistogram {
	if store, exists := e.stores[id]; exists {
		return store.(*PerfHistogram)
	}
	return nil
}

// ReadyMetrics returns the set of ready stores (ie. having data and a passed
// period). This operation blocks metrics stores operations and should be
// wisely used.
func (e *Engine) ReadyMetrics() (expiredMetrics map[string]ReadyStore) {
	expiredMetrics = make(map[string]ReadyStore)
	for id, s := range e.stores {
		if s.Ready() {
			for _, ready := range s.Flush() {
				expiredMetrics[id] = ready
			}
		}
	}
	if len(expiredMetrics) == 0 {
		return nil
	}
	return expiredMetrics
}

type ReadyPerfHistogram struct {
	*ReadyTimeHistogram
	max        float64
	base, unit float64
}

func (s *ReadyPerfHistogram) Unit() float64 { return s.unit }
func (s *ReadyPerfHistogram) Base() float64 { return s.base }
func (s *ReadyPerfHistogram) Max() float64  { return s.max }

type StoreMap map[interface{}]*int64
type ReadyStoreMap map[interface{}]int64

type MaxMetricsStoreLengthError struct {
	MaxLen uint
}

func (e MaxMetricsStoreLengthError) Error() string {
	return fmt.Sprintf("new metrics store key dropped as the metrics store has reached its maximum length `%d`", e.MaxLen)
}

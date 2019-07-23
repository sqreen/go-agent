// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

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
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

// Engine manages the metrics stores in oder to create new one, and to poll
// the existing ones. Engine's methods are not thread-safe and designed to be
// used by a single goroutine.
type Engine struct {
	logger             plog.DebugLogger
	stores             map[string]*Store
	maxMetricsStoreLen uint
}

func NewEngine(logger plog.DebugLogger, maxMetricsStoreLen uint) *Engine {
	return &Engine{
		logger:             logger,
		stores:             make(map[string]*Store),
		maxMetricsStoreLen: maxMetricsStoreLen,
	}
}

// NewStore creates and registers a new metrics store.
func (e *Engine) NewStore(id string, period time.Duration) *Store {
	store := newStore(period, e.maxMetricsStoreLen)
	e.stores[id] = store
	return store
}

// ReadyMetrics returns the set of ready stores (ie. having data and a passed
// period). This operation blocks metrics stores operations and should be
// wisely used.
func (e *Engine) ReadyMetrics() (expiredMetrics map[string]*ReadyStore) {
	expiredMetrics = make(map[string]*ReadyStore)
	for id, s := range e.stores {
		if s.Ready() {
			ready := s.Flush()
			expiredMetrics[id] = ready
			e.logger.Debugf("metrics: store `%s` ready with `%d` entries", id, len(ready.Metrics()))
		}
	}
	if len(expiredMetrics) == 0 {
		return nil
	}
	return expiredMetrics
}

// Store is a metrics store optimized for write accesses to already existing
// keys (cf. Add). It has a period of time after which the data is considered
// ready to be retrieved. An empty store is never considered ready and the
// deadline is computed when the first value is inserted.
type Store struct {
	// Map of comparable types to uint64 pointers.
	set  StoreMap
	lock sync.RWMutex
	// Next deadline, computed when the first value is inserted.
	deadline time.Time
	// Minimum time duration the data should be kept.
	period time.Duration
	// Maximum map length. New keys are dropped when reached.
	// The length is unlimited when 0.
	maxLen uint
}

type StoreMap map[interface{}]*uint64
type ReadyStoreMap map[interface{}]uint64

func newStore(period time.Duration, maxLen uint) *Store {
	return &Store{
		set:    make(StoreMap),
		period: period,
		maxLen: maxLen,
	}
}

type MaxMetricsStoreLengthError struct {
	MaxLen uint
}

func (e MaxMetricsStoreLengthError) Error() string {
	return fmt.Sprintf("new metrics store key dropped as the metrics store has reached its maximum length `%d`", e.MaxLen)
}

// Add delta to the given key, inserting it if it doesn't exist. This method
// is thread-safe and optimized for updating existing key which is lock-free
// when not concurrently retrieving (method `Flush()`) or inserting a new key.
func (s *Store) Add(key interface{}, delta uint64) error {
	// Avoid panic-ing by checking the key type is not nil and comparable.
	if key == nil {
		return sqerrors.New("unexpected key value `nil`")
	} else if !reflect.TypeOf(key).Comparable() {
		return sqerrors.Errorf("unexpected non-comparable type `%T`", key)
	}

	// Fast hot path: concurrently updating the value of an existing key.
	// Lock the store for reading only.
	s.lock.RLock()
	// Lookup the value
	value, exists := s.set[key]
	if exists {
		// The key already exists.
		// Atomically update the value.
		// This update operation can be therefore done concurrently.
		atomic.AddUint64(value, delta)
		// It is important to do it in this write-safe section that is mutually
		// exclusive with Flush() which replaces the store's map using Lock().
	}
	// Unlock the store
	s.lock.RUnlock()

	// Slow path: the key does not exist
	if !exists {
		// Exclusively lock the store
		s.lock.Lock()
		defer s.lock.Unlock()
		// Check again in case the value has been inserted while getting here.
		value, exists = s.set[key]
		if exists {
			// The value was inserted by another concurrent goroutine.
			// We can update the value without atomic operation as we exclusively
			// have the lock.
			*value += delta
			// Note that this is not possible to unlock and perform the atomic
			// operation because of possible concurrent `Flush()`.
		} else {
			if l := len(s.set); l == 0 {
				// Set the deadline when the first valuMaxMetricsStoreLengthe inserted into the metrics store
				s.deadline = time.Now().Add(s.period)
			} else if s.maxLen > 0 && uint(l) >= s.maxLen {
				// The maximum length is reached - no more new insertions are allowed
				return MaxMetricsStoreLengthError{MaxLen: s.maxLen}
			}
			// The value still doesn't exist and we need to insert it into the store's
			// map.
			value := delta
			s.set[key] = &value
		}
	}

	return nil
}

// Flush returns the stored data and the corresponding time window the data was
// held. It should be used when the store is `Ready()`. This method is
// thead-safe.
func (s *Store) Flush() (flushed *ReadyStore) {
	// Read current time before swapping the stores in order to avoid making it in
	// the critical-section. Reading it before is important in order to get
	// old.finish <= new.start.
	now := time.Now()

	// Exclusively lock the store in order to get the values and replace it.
	s.lock.Lock()
	oldMap := s.set
	startedAt := s.deadline.Add(-s.period)
	// Create a new map with the same capacity as the old one to avoid allocation
	// time when still used the same way after the flush.
	s.set = make(StoreMap, len(oldMap))
	s.deadline = time.Time{} // time.Time zero value
	// Unlock the store which is ready to be used again by concurrent goroutines.
	s.lock.Unlock()

	// Compute the map of values getting rid of the pointers (less GC-pressure).
	readyMap := make(ReadyStoreMap, len(oldMap))
	for k, v := range oldMap {
		readyMap[k] = *v
	}
	return &ReadyStore{
		set:    readyMap,
		start:  startedAt,
		finish: now,
	}
}

// Ready returns true when the store has values and the period passed.
// This method is thread-safe. Note that the atomic operation
// "Ready() + Flush()" doesn't exist, they should therefore be used by a single
// "flusher" goroutine. The locking of `Ready()` is indeed weaker than `Flush()`
// as it only lock the store for reading in order to avoid blocking other
// concurrent updates.
func (s *Store) Ready() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return !s.deadline.IsZero() && time.Now().After(s.deadline)
}

// ReadyStore provides methods to get the values and the time window.
type ReadyStore struct {
	set           ReadyStoreMap
	start, finish time.Time
}

func (s *ReadyStore) Start() time.Time {
	return s.start
}

func (s *ReadyStore) Finish() time.Time {
	return s.finish
}

func (s *ReadyStore) Metrics() ReadyStoreMap {
	return s.set
}

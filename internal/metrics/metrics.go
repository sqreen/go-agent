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
	"math"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

// Engine manages the metrics stores in oder to create new one, and to poll
// the existing ones. Engine's methods are not thread-safe and designed to be
// used by a single goroutine.
type Engine struct {
	logger             plog.DebugLogger
	stores             map[string]Store
	maxMetricsStoreLen uint
}

type Store interface {
	Ready() bool
	Flush() ReadyStore
}

type ReadyStore interface {
	Start() time.Time
	Finish() time.Time
	Metrics() ReadyStoreMap
}

func NewEngine(logger plog.DebugLogger, maxMetricsStoreLen uint) *Engine {
	return &Engine{
		logger:             logger,
		stores:             make(map[string]Store),
		maxMetricsStoreLen: maxMetricsStoreLen,
	}
}

// GetSumStore creates and registers a new metrics store when it does not exist. It
// returns the existing one otherwise.
func (e *Engine) GetSumStore(id string, period time.Duration) *SumStore {
	// TODO: rwlock e.stores
	if store, exists := e.stores[id]; exists {
		return store.(*SumStore)
	}
	store := newStore(period, e.maxMetricsStoreLen)
	e.stores[id] = store
	return store
}

func (e *Engine) GetBinningStore(id string, unit, base float64, period time.Duration) (*BinningStore, error) {
	if store, exists := e.stores[id]; exists {
		return store.(*BinningStore), nil
	}

	if unit <= 0.0 {
		return nil, sqerrors.Errorf("unexpected binning unit value `%f`", unit)
	}
	if base <= 1.0 {
		return nil, sqerrors.Errorf("unexpected binning base value `%f`", base)
	}

	store := newBinningStore(period, unit, base, e.maxMetricsStoreLen)
	e.stores[id] = store
	return store, nil
}

// ReadyMetrics returns the set of ready stores (ie. having data and a passed
// period). This operation blocks metrics stores operations and should be
// wisely used.
func (e *Engine) ReadyMetrics() (expiredMetrics map[string]ReadyStore) {
	expiredMetrics = make(map[string]ReadyStore)
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

// SumStore is a metrics store optimized for write accesses to already existing
// keys (cf. Add). It has a period of time after which the data is considered
// ready to be retrieved. An empty store is never considered ready and the
// deadline is computed when the first value is inserted.
type SumStore struct {
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

type BinningStore struct {
	unit, base float64
	invLogBase float64
	subParcel  float64
	// The max value is a uint64 value so that we can assign it with a lock-less
	// operation using atomic integer operations.
	maxValue int64
	// The store of bins and their values. The existing sum store is suitable.
	sumStore *SumStore
	// Lock to atomically flush the store
	rwlock sync.RWMutex
}

func newBinningStore(period time.Duration, unit, base float64, maxStoreLen uint) *BinningStore {
	sumStore := newStore(period, maxStoreLen)
	logBase := math.Log(base)
	logUnit := math.Log(unit)

	return &BinningStore{
		unit:       unit,
		base:       base,
		invLogBase: 1 / logBase,
		subParcel:  logUnit / logBase,
		sumStore:   sumStore,
	}
}

func (s *BinningStore) Ready() bool {
	return s.sumStore.Ready()
}

type ReadyBinningStore struct {
	*ReadySumStore
	max        int64
	base, unit float64
}

func (s *ReadyBinningStore) Unit() float64 { return s.unit }
func (s *ReadyBinningStore) Base() float64 { return s.base }
func (s *ReadyBinningStore) Max() int64    { return s.max }

func (s *BinningStore) Flush() ReadyStore {
	s.rwlock.Lock()
	defer s.rwlock.Unlock()

	max := atomic.LoadInt64(&s.maxValue)
	atomic.StoreInt64(&s.maxValue, 0)
	sumStore := s.sumStore.Flush().(*ReadySumStore)
	return &ReadyBinningStore{
		ReadySumStore: sumStore,
		max:           max,
		base:          s.base,
		unit:          s.unit,
	}
}

func (s *BinningStore) Add(v float64) error {
	s.rwlock.RLock()
	defer s.rwlock.RUnlock()

	// Updating the max value only with the integer part, assuming its unit is
	// enough to be interesting (eg. milliseconds vs microseconds).
	s.updateMax(int64(math.Floor(v)))
	return s.sumStore.Add(s.bin(v), 1)
}

func (s *BinningStore) bin(v float64) (bin uint64) {
	if v < s.unit {
		return 1
	}
	r := math.Floor(math.Log(v)*s.invLogBase - s.subParcel)
	sqassert.True(r > 0 && !math.IsNaN(r) && math.IsInf(r, 0))
	return 2 + uint64(r)
}

// Lock-less update of the max value using a compare-and-swap loop.
func (s *BinningStore) updateMax(v int64) {
	// Load the current max value and try to update it when the new value is
	// bigger by using the CAS operation. When successfully swapped, we can return
	// from the function. But if not, we need to retry by reloading the current
	// value and checking again if the new value is the new max, and retry again
	// if necessary.
	for {
		current := atomic.LoadInt64(&s.maxValue)
		if v <= current {
			break
		}
		if swapped := atomic.CompareAndSwapInt64(&s.maxValue, current, v); swapped {
			break
		}
	}
}

type StoreMap map[interface{}]*int64
type ReadyStoreMap map[interface{}]int64

func newStore(period time.Duration, maxLen uint) *SumStore {
	return &SumStore{
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
func (s *SumStore) Add(key interface{}, delta int64) error {
	// Avoid panic-ing by checking the key type is not nil and comparable.
	if key == nil {
		return sqerrors.New("unexpected key value `nil`")
	}
	if !reflect.TypeOf(key).Comparable() {
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
		atomic.AddInt64(value, delta)
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
				// Set the deadline when this is the first value inserted into the store
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
func (s *SumStore) Flush() (flushed ReadyStore) {
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
	return &ReadySumStore{
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
func (s *SumStore) Ready() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return !s.deadline.IsZero() && time.Now().After(s.deadline)
}

// ReadySumStore provides methods to get the values and the time window.
type ReadySumStore struct {
	set           ReadyStoreMap
	start, finish time.Time
}

func (s *ReadySumStore) Start() time.Time {
	return s.start
}

func (s *ReadySumStore) Finish() time.Time {
	return s.finish
}

func (s *ReadySumStore) Metrics() ReadyStoreMap {
	return s.set
}

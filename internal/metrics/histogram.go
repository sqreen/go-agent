// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package metrics

import (
	"math"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

// TimeHistogram is a metrics store optimized for write accesses to already
// existing keys. The data is stored per time bucket according to the current
// time and the configured time period. Time buckets are simply the current time
// aligned on the period (eg. period 1' implies time buckets 0', 1', 2', etc.).
// A TimeHistogram is ready as soon as it stores data and at least its period
// has passed.
type TimeHistogram struct {
	// Map of time buckets to a pointer to a sync.Map of comparable types to
	// int64 pointers.
	buckets sync.Map

	// Minimum time duration the data should be kept.
	period time.Duration

	// Global RWMutex for exclusively locking the flush operation. This ensures
	// the overall time coherence so that no writer from the past can happen after
	// the flush has been done.
	flushLock sync.RWMutex

	// Once lock to update the start and deadline.
	once sync.Once

	// start time, aligned on the period duration, from which the first value was
	// added.
	start time.Time
}

func NewTimeHistogram(period time.Duration, maxLen uint) *TimeHistogram {
	return &TimeHistogram{
		period: period,
	}
}

// Add delta to the given key, inserting it if it doesn't exist. This method
// is thread-safe and optimized for updating existing keys.
func (s *TimeHistogram) Add(key interface{}, delta int64) error {
	// Avoid panic-ing by checking the key type is not nil and comparable.
	if key == nil {
		return sqerrors.New("unexpected key value `nil`")
	}
	if !reflect.TypeOf(key).Comparable() {
		return sqerrors.Errorf("unexpected non-comparable type `%T`", key)
	}

	s.flushLock.RLock()
	defer s.flushLock.RUnlock()

	_, err := s.add(key, delta)

	return err
}

func (s *TimeHistogram) add(key interface{}, delta int64) (uint64, error) {
	s.once.Do(func() {
		now := time.Now().Truncate(s.period)
		s.start = now
	})

	bucket := s.bucket()

	var store *sync.Map
	if v, _ := s.buckets.Load(bucket); v != nil {
		store = v.(*sync.Map)
	} else {
		store = &sync.Map{}
		if actual, loaded := s.buckets.LoadOrStore(bucket, store); loaded {
			store = actual.(*sync.Map)
		}
	}

	// Fast hot path: concurrently updating the value of an existing key.
	// Atomically update the value.
	// This update operation can be therefore done concurrently.
	actual, loaded := store.Load(key)
	if !loaded {
		actual, loaded = store.LoadOrStore(key, &delta)
	}

	if loaded {
		// Atomically update the value.
		sum := actual.(*int64)
		atomic.AddInt64(sum, delta)
	} // else this was the first value added

	return bucket, nil
}

// Flush returns the stored data and the corresponding time window the data was
// held. It should be used when the store is `Ready()`. This method is
// thead-safe.
func (s *TimeHistogram) Flush() (ready []ReadyStore) {
	start, buckets := s.flush()
	return makeReadyTimeHistogram(start, s.period, buckets)
}

func (s *TimeHistogram) flush() (start time.Time, buckets sync.Map) {
	// Exclusively lock the store in order to get the values and replace it.
	// No one else can be adding new data into the store
	s.flushLock.Lock()
	defer s.flushLock.Unlock()
	_, start, buckets = s.flushUnsafe()
	return start, buckets
}

func (s *TimeHistogram) flushUnsafe() (ongoing uint64, start time.Time, buckets sync.Map) {
	// Save the ready histogram and its starting time
	buckets = s.buckets
	start = s.start

	// Load the current time bucket values if any
	var ongoingBucket *sync.Map
	bucket := s.bucket()
	if ongoing, loaded := buckets.Load(bucket); loaded {
		// LoadAndDelete() is available from go1.15 - we can't use it for now
		buckets.Delete(bucket)
		ongoingBucket = ongoing.(*sync.Map)
	}

	// Reset the histogram
	s.buckets = sync.Map{}
	s.once = sync.Once{}
	if ongoingBucket != nil {
		// The current bucket is ongoing: set the start time and bucket list
		// accordingly.
		s.start = start.Add(time.Duration(bucket) * s.period)
		bucket = 0
		s.buckets.Store(bucket, ongoingBucket)
		ongoing = bucket
	} else {
		s.start = time.Time{}
	}

	return ongoing, start, buckets
}

func makeReadyTimeHistogram(start time.Time, period time.Duration, buckets sync.Map) (ready []ReadyStore) {
	buckets.Range(func(k, v interface{}) bool {
		bucket := k.(uint64)
		st := start.Add(time.Duration(bucket) * period)

		readyMap := make(ReadyStoreMap)
		v.(*sync.Map).Range(func(k, v interface{}) bool {
			readyMap[k] = *v.(*int64)
			return true
		})

		ready = append(ready, &ReadyTimeHistogram{
			timeBucket: bucket,
			set:        readyMap,
			start:      st,
			finish:     st.Add(period),
		})
		return true
	})

	sort.Slice(ready, func(i, j int) bool {
		return ready[i].Start().Before(ready[j].Start())
	})

	return ready
}

// Ready returns true when the store has values and the period passed.
// This method is thread-safe. Note that the atomic operation
// "Ready() + Flush()" doesn't exist, they should therefore be used by a single
// "flusher" goroutine. The locking of `Ready()` is indeed weaker than `Flush()`
// as it only lock the store for reading in order to avoid blocking other
// concurrent updates.
func (s *TimeHistogram) Ready() bool {
	s.flushLock.RLock()
	defer s.flushLock.RUnlock()
	return !s.start.IsZero() && time.Since(s.start) >= s.period
}

func (s *TimeHistogram) bucket() uint64 {
	return uint64(time.Since(s.start) / s.period)
}

// ReadyTimeHistogram provides methods to get the values and the time window.
type ReadyTimeHistogram struct {
	set           ReadyStoreMap
	start, finish time.Time
	timeBucket    uint64
}

func (s *ReadyTimeHistogram) Start() time.Time {
	return s.start
}

func (s *ReadyTimeHistogram) Finish() time.Time {
	return s.finish
}

func (s *ReadyTimeHistogram) Metrics() ReadyStoreMap {
	return s.set
}

// PerfHistogram is a performance monitoring histogram storing performance
// metrics per time and per "performance bucket". It is based on a
// TimeHistogram storing those performance buckets.
type PerfHistogram struct {
	unit, base float64
	invLogBase float64
	subParcel  float64

	// The time histogram of performance buckets and their values.
	timeHistogram *TimeHistogram

	// Separate simplified time histogram of max values. It follows the same
	// number of time buckets as the performance buckets'
	maxValues sync.Map
}

func NewPerfHistogram(period time.Duration, unit, base float64, maxStoreLen uint) (*PerfHistogram, error) {
	if unit <= 0.0 {
		return nil, sqerrors.Errorf("unexpected binning unit value `%f`", unit)
	}
	if base <= 1.0 {
		return nil, sqerrors.Errorf("unexpected binning base value `%f`", base)
	}

	sumStore := NewTimeHistogram(period, maxStoreLen)
	logBase := math.Log(base)
	logUnit := math.Log(unit)

	return &PerfHistogram{
		unit:          unit,
		base:          base,
		invLogBase:    1 / logBase,
		subParcel:     logUnit / logBase,
		timeHistogram: sumStore,
	}, nil
}

func (s *PerfHistogram) Ready() bool {
	return s.timeHistogram.Ready()
}

func (s *PerfHistogram) Add(v float64) error {
	s.timeHistogram.flushLock.RLock()
	defer s.timeHistogram.flushLock.RUnlock()

	perfBucket := s.bucket(v)
	// TODO: pass the current time now to use
	timeBucket, err := s.timeHistogram.add(perfBucket, 1)
	if err != nil {
		return err
	}

	s.updateMax(timeBucket, v)
	return nil
}

func (s *PerfHistogram) bucket(v float64) (bin uint64) {
	if v < s.unit {
		return 1
	}
	r := math.Floor(math.Log(v)*s.invLogBase - s.subParcel)
	sqassert.True(r >= 0 && !math.IsNaN(r) && !math.IsInf(r, 0))
	return 2 + uint64(r)
}

// Lock-less update of the max value using a compare-and-swap loop.
func (s *PerfHistogram) updateMax(timeBucket uint64, v float64) {
	// Fast path: try to load the max pointer first. This allows to only use
	// LoadOrStore and its pointer allocation for every call to updateMax.
	maxBitsPtrFace, loaded := s.maxValues.Load(timeBucket)
	if !loaded {
		// Slow path
		maxBits := math.Float64bits(v)
		maxBitsPtr := &maxBits
		// Note this involves an allocation we want to avoid when not necessary,
		// hence the first use of Load() alone to.
		maxBitsPtrFace, loaded = s.maxValues.LoadOrStore(timeBucket, maxBitsPtr)
		if !loaded {
			// First value
			return
		}
	}
	maxBitsPtr := maxBitsPtrFace.(*uint64)

	// Load the current max value and try to update it when the new value is
	// bigger by using the CAS operation. When successfully swapped, we can return
	// from the function. But if not, we need to retry by reloading the current
	// value and checking again if the new value is the new max, and retry again
	// if necessary.
	vBits := math.Float64bits(v)
	for {
		maxBits := atomic.LoadUint64(maxBitsPtr)
		max := math.Float64frombits(maxBits)
		if v <= max {
			// This value `v` is smaller than the current max value.
			return
		}

		// This value `v` is greater than the current max value. Therefore, we try
		// to CAS it.
		if swapped := atomic.CompareAndSwapUint64(maxBitsPtr, maxBits, vBits); swapped {
			// Successfully swapped
			break
		}

		// Not swapped - retry everything
	}
}

func (s *PerfHistogram) Flush() (ready []ReadyStore) {
	start, timeBuckets, maxValues := s.flush()

	timeHist := makeReadyTimeHistogram(start, s.timeHistogram.period, timeBuckets)

	for _, timeHist := range timeHist {
		timeHist := timeHist.(*ReadyTimeHistogram)
		v, ok := maxValues.Load(timeHist.timeBucket)
		sqassert.True(ok)
		// TODO: write a helper func to access max values
		max := math.Float64frombits(*v.(*uint64))

		ready = append(ready, &ReadyPerfHistogram{
			ReadyTimeHistogram: timeHist,
			max:                max,
			base:               s.base,
			unit:               s.unit,
		})

	}
	return
}

func (s *PerfHistogram) flush() (start time.Time, timeBuckets, maxValuesTimeBucket sync.Map) {
	s.timeHistogram.flushLock.Lock()
	defer s.timeHistogram.flushLock.Unlock()

	ongoing, start, timeBuckets := s.timeHistogram.flushUnsafe()
	maxValuesTimeBucket = s.maxValues

	if ongoing > 0 {
		v, ok := maxValuesTimeBucket.Load(ongoing)
		sqassert.True(ok)
		maxValuesTimeBucket.Delete(ongoing)

		s.maxValues.Store(0, v)
	}

	s.maxValues = sync.Map{}

	return start, timeBuckets, maxValuesTimeBucket
}

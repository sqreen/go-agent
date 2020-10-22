// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package metrics_test

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/stretchr/testify/require"
)

// The time period must be long enough so that the following sleeps should
// work on any OS (given the fact sleeping is actually "sleep at least").
const (
	MinTestPeriod = 500 * time.Millisecond
	MaxStoreLen   = 100000
)

func TestTimeHistogram(t *testing.T) {
	t.Run("sum store usage", func(t *testing.T) {
		t.Run("empty stores are never ready", func(t *testing.T) {
			store := metrics.NewTimeHistogram(time.Microsecond, MaxStoreLen)
			require.False(t, store.Ready())
			time.Sleep(time.Microsecond)
			require.False(t, store.Ready())
		})

		t.Run("non-empty stores get ready starting as soon as a value was added", func(t *testing.T) {
			period := MinTestPeriod
			store := metrics.NewTimeHistogram(period, MaxStoreLen)

			// Empty stores shouldn't be ready
			require.False(t, store.Ready())

			// Should be still not ready because no values were added
			time.Sleep(period)
			require.False(t, store.Ready())

			// Now add a value
			testStartedAt := time.Now()
			require.NoError(t, store.Add("key 1", 1))
			testFinishedAt := time.Now()

			// Wait for a full period to make it available
			time.Sleep(period)

			// The store should be now expired
			require.True(t, store.Ready())

			// Flushing the store should give the ready result
			ready := store.Flush()
			// The old store should have the stored values
			checkTimeHistogram(t, period, testStartedAt, testFinishedAt, metrics.ReadyStoreMap{"key 1": 1}, ready)

			// The store cannot be ready without new values
			require.False(t, store.Ready())
			time.Sleep(period)
			require.False(t, store.Ready())
		})

		t.Run("Flush", func(t *testing.T) {
			period := MinTestPeriod
			store := metrics.NewTimeHistogram(period, MaxStoreLen)

			test1StartedAt := time.Now()

			store.Add("k1", 1)
			time.Sleep(period)
			require.True(t, store.Ready())

			// Add new values after the period, meaning they will go in another time
			// bucket than the previous k1
			store.Add("k2", 2)
			store.Add("k1", 1)

			// Flush the store to see if the ongoing bucket is correctly handled
			ready1 := store.Flush()
			test1FinishedAt := time.Now()
			require.NotEmpty(t, ready1)

			// Add new values that should go into the current bucket
			store.Add("k2", 3)
			store.Add("k3", 3)

			// Wait until it becomes ready
			time.Sleep(period)
			require.True(t, store.Ready())

			ready2 := store.Flush()
			test2FinishedAt := time.Now()
			require.NotEmpty(t, ready2)

			checkTimeHistogram(
				t,
				period,
				test1StartedAt,
				test1FinishedAt,
				metrics.ReadyStoreMap{
					"k1": 1,
				},
				ready1)

			checkTimeHistogram(
				t,
				period,
				test1FinishedAt,
				test2FinishedAt,
				metrics.ReadyStoreMap{
					"k1": 1,
					"k2": 5,
					"k3": 3,
				},
				ready2)
		})

		t.Run("adding values to a store that is ready is possible", func(t *testing.T) {
			period := MinTestPeriod
			store := metrics.NewTimeHistogram(period, MaxStoreLen)

			// Store is empty so it can't be ready
			require.False(t, store.Ready())

			testStartedAt := time.Now()

			// Add a new value
			require.NoError(t, store.Add("key 1", 1))

			// Wait for one period
			time.Sleep(period)
			require.True(t, store.Ready())

			// Add new values even if ready
			require.NoError(t, store.Add("key 1", 1))
			require.NoError(t, store.Add("key 2", 33))
			require.NoError(t, store.Add("key 3", 33))
			require.NoError(t, store.Add("key 3", 1))

			testFinishedAt := time.Now()

			// Wait for a new period so that every value is ready
			time.Sleep(period)
			require.True(t, store.Ready())

			ready := store.Flush()
			checkTimeHistogram(
				t,
				period,
				testStartedAt,
				testFinishedAt,
				metrics.ReadyStoreMap{
					"key 1": 2,
					"key 2": 33,
					"key 3": 34,
				},
				ready)
		})

		t.Run("key types", func(t *testing.T) {
			period := MinTestPeriod
			store := metrics.NewTimeHistogram(period, MaxStoreLen)

			t.Run("non comparable key types are not allowed and do not panic", func(t *testing.T) {
				type Struct2 struct {
					a int
					b string
					c float32
					d []byte
				}

				require.NotPanics(t, func() {
					require.Error(t, store.Add([]byte("no slices"), 1))
					require.Error(t, store.Add(map[string]string{"a": "b", "c": "d"}, 21))
					require.Error(t, store.Add(Struct2{
						a: 33,
						b: "string",
						c: 4.815162342,
						d: []byte("no slice"),
					}, 1))
				})
			})

			t.Run("comparable key types are allowed and do not panic", func(t *testing.T) {
				type Struct struct {
					a int
					b string
					c float32
					d [33]byte
				}

				type T1 struct{}
				type T2 struct{}
				var v1, v2 interface{} = T1{}, T2{}

				ptr := &Struct{}

				testStartedAt := time.Now()

				require.NotPanics(t, func() {
					require.NoError(t, store.Add("string", 1))
					require.NoError(t, store.Add(T1{}, 1))
					require.NoError(t, store.Add(v1, 3))
					require.NoError(t, store.Add(T2{}, 3))
					require.NoError(t, store.Add(v2, 5))
					require.NoError(t, store.Add("string", 1))
					require.NoError(t, store.Add("string", 1))
					require.NoError(t, store.Add(33, 1))
					require.NoError(t, store.Add(Struct{
						a: 33,
						b: "string",
						c: 4.815162342,
						d: [33]byte{},
					}, 1))
					require.NoError(t, store.Add(ptr, 1))
					// Nil is comparable but not allowed
					require.Error(t, store.Add(nil, 1))
				})

				testFinishedAt := time.Now()

				time.Sleep(period)
				require.True(t, store.Ready())
				old := store.Flush()
				checkTimeHistogram(
					t,
					period,
					testStartedAt,
					testFinishedAt,
					metrics.ReadyStoreMap{
						"string": 3,
						33:       1,
						Struct{
							a: 33,
							b: "string",
							c: 4.815162342,
							d: [33]byte{},
						}: 1,
						ptr:  1,
						v1:   4,
						T2{}: 8,
					},
					old)
			})

			t.Run("error key type for error aggregation", func(t *testing.T) {
				period := MinTestPeriod
				errs := metrics.NewTimeHistogram(period, MaxStoreLen)

				// Prepare some errors
				err1 := errors.New("error 1")
				err2 := errors.New("error 2")
				wrap1 := sqerrors.Wrap(err1, "wrap 1")
				wrap2 := sqerrors.Wrap(wrap1, "wrap 2")

				// Add them
				testStartedAt := time.Now()
				require.NoError(t, errs.Add(err1, 1))
				require.NoError(t, errs.Add(err2, 1))
				require.NoError(t, errs.Add(err1, 1))
				require.NoError(t, errs.Add(wrap1, 1))
				require.NoError(t, errs.Add(wrap2, 1))
				require.NoError(t, errs.Add(wrap1, 1))
				testFinishedAt := time.Now()

				time.Sleep(period)
				require.True(t, errs.Ready())
				ready := errs.Flush()

				checkTimeHistogram(
					t,
					period,
					testStartedAt,
					testFinishedAt,
					metrics.ReadyStoreMap{
						err1:  2,
						err2:  1,
						wrap1: 2,
						wrap2: 1,
					},
					ready)
			})
		})

		t.Run("time bucketing", func(t *testing.T) {
			t.Run("", func(t *testing.T) {
				// Use a large-enough period so that we can check the bucketing
				period := MinTestPeriod

				store := metrics.NewTimeHistogram(period, MaxStoreLen)

				// Add one value per period to enforce the number of buckets
				expectedBucketCount := 10
				startAdding := time.Now()
				for i := 0; i < expectedBucketCount; i++ {
					require.NoError(t, store.Add(i, int64(i)))
					time.Sleep(period)
				}
				stopAdding := time.Now()

				require.True(t, store.Ready())

				// Flush the store
				ready := store.Flush()
				require.Len(t, ready, expectedBucketCount)

				// Align the expected boundaries with the period
				startAdding = startAdding.Truncate(period)
				stopAdding = stopAdding.Truncate(period).Add(period)

				// Check the first and last values' times:
				// - Start should be <= startAdding
				start0 := ready[0].Start()
				require.Truef(t, startAdding.Equal(start0) || startAdding.Before(start0), "expected at least=%s but got %s", startAdding, start0)

				// - Finish should be >= stopAdding
				finishN := ready[len(ready)-1].Finish()
				require.True(t, stopAdding.Equal(finishN) || stopAdding.After(finishN))

				prevFinish := startAdding
				for i, ready := range ready {
					// prevFinish <= start
					start := ready.Start()
					require.Truef(t, prevFinish.Equal(start) || prevFinish.Before(start), "expected at least=%s but got %s", prevFinish, start)

					// finish - start == period
					finish := ready.Finish()
					require.True(t, finish.Sub(start) == period)

					metrics := ready.Metrics()

					// One value per bucket expected
					require.Len(t, metrics, 1)

					require.Equal(t, int64(i), metrics[i])

					prevFinish = finish
				}

				require.Truef(t, prevFinish.Equal(stopAdding) || prevFinish.Before(stopAdding), "expected at least=%s but got %s")
			})

			t.Run("", func(t *testing.T) {
				// Use a large-enough period so that we can check the bucketing
				period := 2 * time.Second

				store := metrics.NewTimeHistogram(period, MaxStoreLen)

				// Add one value per period to enforce the number of buckets
				expectedMetricsCount := 10

				startAdding := time.Now()
				for i := 0; i < expectedMetricsCount; i++ {
					require.NoError(t, store.Add(i, int64(i)))
				}

				require.False(t, store.Ready())

				// Flush the store anyway
				ready := store.Flush()
				require.Len(t, ready, 0)

				for i := 0; i < expectedMetricsCount; i++ {
					require.NoError(t, store.Add(i, int64(i)))
				}
				stopAdding := time.Now()

				time.Sleep(period)
				require.True(t, store.Ready())

				ready = store.Flush()
				require.Len(t, ready, 1)

				// Align the expected boundaries with the period
				startAdding = startAdding.Truncate(period)
				stopAdding = stopAdding.Truncate(period).Add(period)

				// Check the first and last values' times:
				// - Start should be <= startAdding
				start := ready[0].Start()
				require.Truef(t, startAdding.Equal(start) || startAdding.Before(start), "expected at least=%s but got %s", startAdding, start)

				// - Finish should be >= stopAdding
				finish := ready[0].Finish()
				require.True(t, stopAdding.Equal(finish) || stopAdding.After(finish))

				metrics := ready[0].Metrics()
				require.Len(t, metrics, expectedMetricsCount)

				for i := 0; i < len(metrics); i++ {
					require.Equal(t, int64(2*i), metrics[i])
				}
			})
		})
	})

	t.Run("one reader - 8000 writers", func(t *testing.T) {
		// Create a store that will be checked more often than actually required by
		// its period. So that we cover the case where the store is not always
		// ready.
		period := MinTestPeriod
		store := metrics.NewTimeHistogram(period, MaxStoreLen)

		// The reader will be awaken 4 times per store period so that we stress test
		// the store's `Ready()` method
		readerTicker := time.Tick(period / 4)

		// Signal channel between this test and the reader to tear down the reader
		done := make(chan struct{})

		// Array of metrics flushed by the reader
		var ready []metrics.ReadyStore

		// One reader
		go func() {
			for {
				select {
				case <-readerTicker:
					if store.Ready() {
						ready = append(ready, store.Flush()...)
					}

				case <-done:
					// Wait one more period to get the last metrics
					time.Sleep(5 * period)

					if store.Ready() {
						ready = append(ready, store.Flush()...)
					}

					// Notify we are done and so the data is ready to be checked
					close(done)
					return
				}
			}
		}()

		// Start 8000 writers that will write 1000 times
		nbWriters := 8000
		nbWrites := 1000

		// Create a stopBarrier to signal when all goroutines are done writing
		// their values
		var stopBarrier sync.WaitGroup
		stopBarrier.Add(nbWriters)

		// Synchronize every goroutine with a starting condition
		var startBarrier sync.WaitGroup
		startBarrier.Add(1)

		for n := 0; n < nbWriters; n++ {
			go func() {
				defer stopBarrier.Done() // Signal we are done when returning

				startBarrier.Wait()

				for c := 0; c < nbWrites; c++ {
					if err := store.Add(c, 1); err != nil {
						t.Fatal(err)
					}
				}
			}()
		}

		// Save the test start time to compare it to the first metrics store start
		// time.
		testStartedAt := time.Now().Truncate(period)

		// Unblock the writer goroutines
		startBarrier.Add(-1)
		// Wait for the writer goroutines to be done
		stopBarrier.Wait()
		// Signal the reader they are done
		done <- struct{}{}
		// Wait for the reader to be done
		<-done

		testFinishedAt := time.Now().Truncate(period).Add(period)

		// There should be no more values available
		time.Sleep(period)
		require.False(t, store.Ready())

		expectedMetrics := metrics.ReadyStoreMap{}
		for n := 0; n < nbWrites; n++ {
			expectedMetrics[n] = int64(nbWriters)
		}

		checkTimeHistogram(
			t,
			period,
			testStartedAt,
			testFinishedAt,
			expectedMetrics,
			ready,
		)
	})
}

func TestPerfHistogram(t *testing.T) {
	// TODO: test reset max alg
	t.Run("bucket algorithm", func(t *testing.T) {
		for _, tc := range []struct {
			Base, Unit      float64
			Values          []float64
			ExpectedMetrics metrics.ReadyStoreMap
			ExpectedMax     float64
			ExpectedError   bool
		}{
			{
				Base:            2,
				Unit:            1,
				Values:          []float64{1.0, 0.2, 2.2, 2.0, -0.0},
				ExpectedMetrics: metrics.ReadyStoreMap{uint64(1): 2, uint64(2): 1, uint64(3): 2},
				ExpectedMax:     2.2,
			},

			{
				Base:            2.0,
				Unit:            0.1,
				Values:          []float64{0.001, 0.1, 0.15, 7.0},
				ExpectedMetrics: metrics.ReadyStoreMap{uint64(1): 1, uint64(2): 2, uint64(8): 1},
				ExpectedMax:     7,
			},

			{
				Base:            2.0,
				Unit:            0.1,
				Values:          []float64{150, -10, 110.8946, 250, 192, 195, 154},
				ExpectedMetrics: metrics.ReadyStoreMap{uint64(1): 1, uint64(12): 5, uint64(13): 1},
				ExpectedMax:     250,
			},

			{
				Unit:          -0,
				Base:          42,
				ExpectedError: true,
			},

			{
				Unit:          0,
				Base:          42,
				ExpectedError: true,
			},


			{
				Unit:          42,
				Base:          0.9999999999999999999999999,
				ExpectedError: true,
			},
		} {
			tc := tc
			t.Run("", func(t *testing.T) {
				period := MinTestPeriod
				store, err := metrics.NewPerfHistogram(period, tc.Unit, tc.Base, MaxStoreLen)
				if tc.ExpectedError {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				for _, v := range tc.Values {
					require.NoError(t, store.Add(v))
				}

				time.Sleep(period)

				ready := store.Flush()[0].(*metrics.ReadyPerfHistogram)

				require.Equal(t, tc.ExpectedMetrics, ready.Metrics())
				require.Equal(t, tc.ExpectedMax, ready.Max())
				require.Equal(t, tc.Unit, ready.Unit())
				require.Equal(t, tc.Base, ready.Base())
			})
		}
	})

	t.Run("perf histogram usage", func(t *testing.T) {
		t.Run("empty stores are never ready", func(t *testing.T) {
			store, err := metrics.NewPerfHistogram(time.Microsecond, 10, 10, MaxStoreLen)
			require.NoError(t, err)
			require.False(t, store.Ready())
			time.Sleep(time.Microsecond)
			require.False(t, store.Ready())
		})

		t.Run("non-empty stores get ready starting as soon as a value was added", func(t *testing.T) {
			period := MinTestPeriod
			store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
			require.NoError(t, err)

			// Empty stores shouldn't be ready
			require.False(t, store.Ready())

			// Should be still not ready because no values were added
			time.Sleep(period)
			require.False(t, store.Ready())

			// Now add a value
			testStartedAt := time.Now()
			require.NoError(t, store.Add(1))
			testFinishedAt := time.Now()

			// Wait for a full period to make it available
			time.Sleep(period)

			// The store should be now expired
			require.True(t, store.Ready())

			// Flushing the store should give the ready result
			ready := store.Flush()

			// The old store should have the stored values
			checkPerfHistogram(t, period, testStartedAt, testFinishedAt, metrics.ReadyStoreMap{uint64(1): 1}, ready, 1, 10, 10)

			// The store cannot be ready without new values
			require.False(t, store.Ready())
			time.Sleep(period)
			require.False(t, store.Ready())
		})

		t.Run("Flush", func(t *testing.T) {
			period := MinTestPeriod
			store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
			require.NoError(t, err)

			require.Empty(t, store.Flush())

			test1StartedAt := time.Now()

			store.Add(1)
			time.Sleep(period)
			require.True(t, store.Ready())

			// Add new values after the period, meaning they will go in another time
			// bucket than the previous k1
			store.Add(2)
			store.Add(1)

			// Flush the store to see if the ongoing bucket is correctly handled
			ready1 := store.Flush()
			test1FinishedAt := time.Now()
			require.NotEmpty(t, ready1)

			// Add new values that should go into the current bucket
			store.Add(3)
			store.Add(3)

			// Wait until it becomes ready
			time.Sleep(period)
			require.True(t, store.Ready())

			ready2 := store.Flush()
			test2FinishedAt := time.Now()
			require.NotEmpty(t, ready2)

			checkPerfHistogram(
				t,
				period,
				test1StartedAt,
				test1FinishedAt,
				metrics.ReadyStoreMap{
					uint64(1): 1,
				},
				ready1,
				1,
				10,
				10)

			checkPerfHistogram(
				t,
				period,
				test1FinishedAt,
				test2FinishedAt,
				metrics.ReadyStoreMap{
					uint64(1): 4,
				},
				ready2,
				3,
				10,
				10)
		})

		t.Run("adding values to a store that is ready is possible", func(t *testing.T) {
			period := MinTestPeriod
			store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
			require.NoError(t, err)

			// Store is empty so it can't be ready
			require.False(t, store.Ready())

			testStartedAt := time.Now()

			// Add a new value
			require.NoError(t, store.Add(1))

			// Wait for one period
			time.Sleep(period)
			require.True(t, store.Ready())

			// Add new values even if ready
			require.NoError(t, store.Add(1))
			require.NoError(t, store.Add(33))
			require.NoError(t, store.Add(33))
			require.NoError(t, store.Add(1))

			testFinishedAt := time.Now()

			// Wait for a new period so that every value is ready
			time.Sleep(period)
			require.True(t, store.Ready())

			ready := store.Flush()
			checkPerfHistogram(
				t,
				period,
				testStartedAt,
				testFinishedAt,
				metrics.ReadyStoreMap{
					uint64(1): 3,
					uint64(2): 2,
				},
				ready,
				33,
				10,
				10)
		})

		t.Run("time bucketing", func(t *testing.T) {
			t.Run("", func(t *testing.T) {
				// Use a large-enough period so that we can check the bucketing
				period := MinTestPeriod

				store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
				require.NoError(t, err)

				// Add one value per period to enforce the number of buckets
				expectedBucketCount := 10
				startAdding := time.Now()
				for i := 0; i < expectedBucketCount; i++ {
					require.NoError(t, store.Add(float64(i)))
					time.Sleep(period)
				}
				stopAdding := time.Now()

				require.True(t, store.Ready())

				// Flush the store
				ready := store.Flush()
				require.Len(t, ready, expectedBucketCount)

				// Align the expected boundaries with the period
				startAdding = startAdding.Truncate(period)
				stopAdding = stopAdding.Truncate(period).Add(period)

				// Check the first and last values' times:
				// - Start should be <= startAdding
				start0 := ready[0].Start()
				require.Truef(t, startAdding.Equal(start0) || startAdding.Before(start0), "expected at least=%s but got %s", startAdding, start0)

				// - Finish should be >= stopAdding
				finishN := ready[len(ready)-1].Finish()
				require.True(t, stopAdding.Equal(finishN) || stopAdding.After(finishN))

				prevFinish := startAdding
				for _, ready := range ready {
					// prevFinish <= start
					start := ready.Start()
					require.Truef(t, prevFinish.Equal(start) || prevFinish.Before(start), "expected at least=%s but got %s", prevFinish, start)

					// finish - start == period
					finish := ready.Finish()
					require.True(t, finish.Sub(start) == period)

					metrics := ready.Metrics()

					// One value per bucket expected
					require.Len(t, metrics, 1)

					require.Equal(t, int64(1), metrics[uint64(1)])

					prevFinish = finish
				}

				require.Truef(t, prevFinish.Equal(stopAdding) || prevFinish.Before(stopAdding), "expected at least=%s but got %s")
			})

			t.Run("", func(t *testing.T) {
				// Use a large-enough period so that we can check the bucketing
				period := 2 * time.Second

				store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
				require.NoError(t, err)

				// Add one value per period to enforce the number of buckets
				nbAdd := 10

				startAdding := time.Now()
				for i := 0; i < nbAdd; i++ {
					require.NoError(t, store.Add(float64(i)))
				}

				require.False(t, store.Ready())

				// Flush the store anyway
				ready := store.Flush()
				require.Len(t, ready, 0)

				for i := 0; i < nbAdd; i++ {
					require.NoError(t, store.Add(float64(i)))
				}
				stopAdding := time.Now()

				time.Sleep(period)
				require.True(t, store.Ready())

				ready = store.Flush()
				require.Len(t, ready, 1)

				// Align the expected boundaries with the period
				startAdding = startAdding.Truncate(period)
				stopAdding = stopAdding.Truncate(period).Add(period)

				// Check the first and last values' times:
				// - Start should be <= startAdding
				start := ready[0].Start()
				require.Truef(t, startAdding.Equal(start) || startAdding.Before(start), "expected at least=%s but got %s", startAdding, start)

				// - Finish should be >= stopAdding
				finish := ready[0].Finish()
				require.True(t, stopAdding.Equal(finish) || stopAdding.After(finish))

				metrics := ready[0].Metrics()
				require.Len(t, metrics, 1)
				require.Equal(t, int64(2*nbAdd), metrics[uint64(1)])
			})
		})
	})

	t.Run("one reader - 8000 writers", func(t *testing.T) {
		// Create a store that will be checked more often than actually required by
		// its period. So that we cover the case where the store is not always
		// ready.
		period := MinTestPeriod
		store, err := metrics.NewPerfHistogram(period, 10, 10, MaxStoreLen)
		require.NoError(t, err)

		// The reader will be awaken 4 times per store period so that we stress test
		// the store's `Ready()` method
		readerTicker := time.Tick(period / 4)

		// Signal channel between this test and the reader to tear down the reader
		done := make(chan struct{})

		// Array of metrics flushed by the reader
		var metricsArray []*metrics.ReadyPerfHistogram

		// One reader
		go func() {
			for {
				select {
				case <-readerTicker:
					if store.Ready() {
						for _, ready := range store.Flush() {
							metricsArray = append(metricsArray, ready.(*metrics.ReadyPerfHistogram))
						}
					}

				case <-done:
					// Wait one more period to get the last metrics
					time.Sleep(2 * period)

					// All goroutines are done, so get the last data left
					if store.Ready() {
						for _, ready := range store.Flush() {
							metricsArray = append(metricsArray, ready.(*metrics.ReadyPerfHistogram))
						}
					}

					// Notify we are done and so the data is ready to be checked
					close(done)
					return
				}
			}
		}()

		// Start 8000 writers that will write 1000 times
		nbWriters := 8000
		nbWrites := 1000

		// Create a stopBarrier to signal when all goroutines are done writing
		// their values
		var stopBarrier sync.WaitGroup
		stopBarrier.Add(nbWriters)

		// Synchronize every goroutine with a starting condition
		var startLock sync.RWMutex
		startLock.Lock()

		for n := 0; n < nbWriters; n++ {
			go func() {
				defer stopBarrier.Done() // Signal we are done when returning

				startLock.RLock()
				defer startLock.RUnlock()

				for c := 0; c < nbWrites; c++ {
					if err := store.Add(float64(100 * c)); err != nil {
						t.Fatal(err)
					}
				}
			}()
		}

		// Save the test start time to compare it to the first metrics store start
		// time.
		testStartedAt := time.Now().Truncate(period)

		// Unblock the writer goroutines
		startLock.Unlock()
		// Wait for the writer goroutines to be done
		stopBarrier.Wait()
		// Signal the reader they are done
		done <- struct{}{}
		// Wait for the reader to be done
		<-done

		testFinishedAt := time.Now().Truncate(period).Add(period)

		// There should be no more values available
		time.Sleep(period)
		require.False(t, store.Ready())

		// Aggregate the ready metrics the reader retrieved and check the previous
		// store finish time is before the current store start time.
		results := make(metrics.ReadyStoreMap)
		prevStoreFinish := testStartedAt.Truncate(period)
		for _, store := range metricsArray {
			for k, v := range store.Metrics() {
				results[k] += v
			}

			require.True(t, prevStoreFinish.Before(store.Start()) || prevStoreFinish.Equal(store.Start()), fmt.Sprint(prevStoreFinish, store.Start()))
			prevStoreFinish = store.Finish()
		}
		require.True(t, prevStoreFinish.Before(testFinishedAt) || prevStoreFinish.Equal(testFinishedAt))

		// Check each writer wrote the expected number of times.
		require.Equal(t, metrics.ReadyStoreMap{
			uint64(1): 8000,
			uint64(2): 8000,
			uint64(3): 72000,
			uint64(4): 720000,
			uint64(5): 7192000,
		}, results)
	})
}

func checkTimeHistogram(t *testing.T, period time.Duration, expectedMinStart, expectedMaxFinish time.Time, expectedMetrics metrics.ReadyStoreMap, actualMetrics []metrics.ReadyStore) {
	expectedMinStart = expectedMinStart.Truncate(period)
	expectedMaxFinish = expectedMaxFinish.Truncate(period).Add(period)

	sum := make(metrics.ReadyStoreMap, len(expectedMetrics))

	prevStoreFinish := expectedMinStart
	for _, ready := range actualMetrics {
		start := ready.Start()
		require.True(t, prevStoreFinish.Before(start) || prevStoreFinish.Equal(start), fmt.Sprint(prevStoreFinish, start))

		for k, v := range ready.Metrics() {
			sum[k] += v
		}

		prevStoreFinish = ready.Finish()
	}
	require.True(t, prevStoreFinish.Before(expectedMaxFinish) || prevStoreFinish.Equal(expectedMaxFinish))

	require.Len(t, sum, len(expectedMetrics))

	for k, v := range sum {
		require.Equal(t, expectedMetrics[k], v)
	}
}

func checkPerfHistogram(t *testing.T, period time.Duration, expectedMinStart, expectedMaxFinish time.Time, expectedMetrics metrics.ReadyStoreMap, actualMetrics []metrics.ReadyStore, expectedMax, expectedUnit, expectedBase float64) {
	checkTimeHistogram(t, period, expectedMinStart, expectedMaxFinish, expectedMetrics, actualMetrics)

	max := math.Inf(-1)
	for _, ready := range actualMetrics {
		ready := ready.(*metrics.ReadyPerfHistogram)

		require.Equal(t, expectedBase, ready.Base())
		require.Equal(t, expectedUnit, ready.Unit())

		if m := ready.Max(); m > max {
			max = m
		}
	}

	require.Equal(t, expectedMax, max)
}

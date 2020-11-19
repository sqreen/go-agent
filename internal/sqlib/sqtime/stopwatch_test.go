// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqtime_test

import (
	"sync"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
	"github.com/stretchr/testify/require"
)

func TestSharedStopWatch(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		watch := sqtime.NewSharedStopWatch()

		// Watch 1: single start/stop
		w1 := watch.Start()
		time.Sleep(time.Millisecond)
		expectedMinDuration := time.Millisecond
		dt := w1.Stop()
		expectedDuration := dt
		require.Equal(t, watch.Duration(), expectedDuration)
		require.GreaterOrEqual(t, int64(dt), int64(time.Millisecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(expectedMinDuration))

		// Watch 2: single start/stop
		w2 := watch.Start()
		time.Sleep(time.Millisecond)
		expectedMinDuration += time.Millisecond
		dt = w2.Stop()
		expectedDuration += dt
		require.Equal(t, watch.Duration(), expectedDuration)
		require.GreaterOrEqual(t, int64(dt), int64(time.Millisecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(expectedMinDuration))

		// Interleaved stopwatches
		// The global duration only increases when every local stop watch is stopped
		globalDuration := watch.Duration()

		// Watch 3: interleaved start/stop
		w3 := watch.Start()

		time.Sleep(5 * time.Millisecond)
		expectedMinDuration += 5 * time.Millisecond

		// Watch 4: interleaved start/stop, stopped before watch 5
		w4 := watch.Start()

		// Watch 5:interleaved start/stop, stopped last
		w5 := watch.Start()

		time.Sleep(5 * time.Millisecond)
		expectedMinDuration += 5 * time.Millisecond

		dt = w4.Stop()
		require.GreaterOrEqual(t, int64(dt), int64(5*time.Millisecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(globalDuration))

		time.Sleep(5 * time.Millisecond)
		expectedMinDuration += 5 * time.Millisecond
		dt = w3.Stop()
		require.GreaterOrEqual(t, int64(dt), int64(2*5*time.Millisecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(globalDuration))

		dt = w5.Stop()
		require.GreaterOrEqual(t, int64(dt), int64(2*5*time.Millisecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(expectedMinDuration))
	})

	t.Run("api checks", func(t *testing.T) {
		watch := sqtime.NewSharedStopWatch()
		local := watch.Start()
		time.Sleep(time.Microsecond)
		dt := local.Stop()
		require.GreaterOrEqual(t, int64(dt), int64(time.Microsecond))
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(time.Microsecond))

		dt2 := local.Stop()
		require.Equal(t, dt, dt2)
	})

	t.Run("shared", func(t *testing.T) {
		var (
			watch        = sqtime.NewSharedStopWatch()
			nbGoroutines = 8000
			startBarrier sync.WaitGroup
			doneBarrier  sync.WaitGroup
		)

		startBarrier.Add(1)
		doneBarrier.Add(nbGoroutines)

		for n := 0; n < nbGoroutines; n++ {
			go func() {
				startBarrier.Wait()

				local := watch.Start()
				time.Sleep(time.Microsecond)
				dt := local.Stop()

				// Avoid using testify assertion helpers to have a faster execution
				if dt < time.Microsecond {
					t.Fatalf("local duration is smaller than the sleep time `%s`", dt)
				}

				doneBarrier.Done()
			}()
		}

		testStartedAt := time.Now()
		startBarrier.Add(-1)
		doneBarrier.Wait()
		testDuration := time.Since(testStartedAt)

		require.GreaterOrEqual(t, int64(testDuration), int64(watch.Duration()))
	})
}

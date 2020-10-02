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
		var watch sqtime.SharedStopWatch
		watch.Start()
		time.Sleep(time.Microsecond)
		watch.Stop()
		require.GreaterOrEqual(t, int64(watch.Duration()), int64(time.Microsecond))
	})

	t.Run("shared", func(t *testing.T) {
		var (
			watch        sqtime.SharedStopWatch
			nbGoroutines = 1000
			startBarrier sync.WaitGroup
			doneBarrier  sync.WaitGroup
		)

		startBarrier.Add(1)
		doneBarrier.Add(nbGoroutines)

		for n := 0; n < nbGoroutines; n++ {
			go func() {
				startBarrier.Wait()

				watch.Start()
				time.Sleep(time.Microsecond)
				watch.Stop()

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

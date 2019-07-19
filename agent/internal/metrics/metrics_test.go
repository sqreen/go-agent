// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package metrics_test

import (
	"fmt"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

var logger = plog.NewLogger(plog.Debug, os.Stderr, 0)

func TestUsage(t *testing.T) {
	engine := metrics.NewEngine(logger)

	t.Run("store usage", func(t *testing.T) {
		t.Run("empty stores are never ready", func(t *testing.T) {
			store := engine.NewStore("id 1", time.Microsecond)
			require.False(t, store.Ready())
			time.Sleep(time.Microsecond)
			require.False(t, store.Ready())
		})

		t.Run("non-empty stores get ready starting as soon as a value was added", func(t *testing.T) {
			store := engine.NewStore("id 1", time.Millisecond)
			require.False(t, store.Ready())
			time.Sleep(time.Millisecond)
			// Should be still not ready because no values were added
			require.False(t, store.Ready())
			// Now add a value
			store.Add("key 1", 1)
			// Should be started but still not expired
			require.False(t, store.Ready())
			time.Sleep(time.Microsecond)
			// Should be still not expired
			require.False(t, store.Ready())
			time.Sleep(time.Millisecond)
			// Now should be expired
			require.True(t, store.Ready())
			// Flushing the store should give the map and "restart" the store
			old := store.Flush()
			require.False(t, store.Ready())
			// Should not be expired while empty
			time.Sleep(time.Millisecond)
			require.False(t, store.Ready())
			// The old store should have the stored values
			require.Equal(t, metrics.ReadyStoreMap{"key 1": 1}, old.Metrics())
			// Adding a new value to the store and then waiting for it to become ready
			// should return the net value
			store.Add("key 2", 3)
			time.Sleep(time.Millisecond)
			require.True(t, store.Ready())
			old = store.Flush()
			require.Equal(t, metrics.ReadyStoreMap{"key 2": 3}, old.Metrics())
		})

		t.Run("adding values to a store that is ready is possible", func(t *testing.T) {
			store := engine.NewStore("id 1", time.Millisecond)
			require.False(t, store.Ready())
			store.Add("key 1", 1)
			time.Sleep(time.Millisecond)
			require.True(t, store.Ready())
			store.Add("key 1", 1)
			store.Add("key 2", 33)
			store.Add("key 3", 33)
			store.Add("key 3", 1)

			require.True(t, store.Ready())
			old := store.Flush()
			require.Equal(t, metrics.ReadyStoreMap{
				"key 1": 2,
				"key 2": 33,
				"key 3": 34,
			}, old.Metrics())
		})

		t.Run("key types", func(t *testing.T) {
			store := engine.NewStore("id 1", time.Millisecond)

			t.Run("non comparable key types are not allowed and do not panic", func(t *testing.T) {
				type Struct2 struct {
					a int
					b string
					c float32
					d []byte
				}

				require.NotPanics(t, func() {
					require.Error(t, store.Add([]byte("no slices"), 1))
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

				ptr := &Struct{}

				require.NotPanics(t, func() {
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

				time.Sleep(time.Millisecond)
				require.True(t, store.Ready())
				old := store.Flush()
				require.Equal(t, metrics.ReadyStoreMap{
					"string": 1,
					33:       1,
					Struct{
						a: 33,
						b: "string",
						c: 4.815162342,
						d: [33]byte{},
					}: 1,
					ptr: 1,
				}, old.Metrics())
			})
		})
	})

	t.Run("one reader - 8000 writers", func(t *testing.T) {
		// Create a store that will be checked more often than actually required by
		// its period. So that we cover the case where the store is not always
		// ready.
		engine := metrics.NewEngine(logger)
		// The reader will be awaken 4 times per store period, so only it will see
		// a ready store only once out of four.
		readerPeriod := time.Microsecond
		metricsStorePeriod := 4 * readerPeriod
		tick := time.Tick(readerPeriod)
		store := engine.NewStore("id", metricsStorePeriod)

		// Signal channel between this test and the reader to tear down the test
		done := make(chan struct{})

		// Array of metrics flushed by the reader
		var metricsArray []*metrics.ReadyStore
		// Time the test finished - it will be compared to the last metrics store
		// finish time
		var finished time.Time

		// One reader
		go func() {
			for {
				select {
				case <-tick:
					if store.Ready() {
						ready := store.Flush()
						metricsArray = append(metricsArray, ready)
					}

				case <-done:
					// All goroutines are done, so read get the last data left
					if ready := store.Flush(); len(ready.Metrics()) > 0 {
						metricsArray = append(metricsArray, ready)
					}
					finished = time.Now()
					// Notify we are done and so the data is ready to be checked
					close(done)
					return
				}
			}
		}()

		// Start 8000 writers that will write 1000 times
		nbWriters := 8000
		nbWrites := 1000

		var startBarrier, stopBarrier sync.WaitGroup
		// Create a start barrier to synchronize every goroutine's launch
		startBarrier.Add(nbWriters)
		// Create a stopBarrier to signal when all goroutines are done writing
		// their values
		stopBarrier.Add(nbWriters)

		for n := 0; n < nbWriters; n++ {
			go func() {
				startBarrier.Wait()      // Sync the starts of the goroutines
				defer stopBarrier.Done() // Signal we are done when returning
				for c := 0; c < nbWrites; c++ {
					_ = store.Add(c, 1)
				}
			}()
		}

		// Save the test start time to compare it to the first metrics store's
		// that should be latter.
		started := time.Now()

		startBarrier.Add(-nbWriters) // Unblock the writer goroutines
		stopBarrier.Wait()           // Wait for the writer goroutines to be done
		done <- struct{}{}           // Signal the reader they are done
		<-done                       // Wait for the reader to be done

		// Make sure there is no data left by sleeping more than needed and checking
		// the store.
		time.Sleep(2 * metricsStorePeriod)
		require.False(t, store.Ready())

		// Aggregate the ready metrics the reader retrieved and check the previous
		// store finish time is before the current store start time.
		results := make(metrics.ReadyStoreMap)
		lastStoreFinish := started
		for _, store := range metricsArray {
			for k, v := range store.Metrics() {
				results[k] += v
			}
			if !lastStoreFinish.IsZero() {
				require.True(t, lastStoreFinish.Before(store.Start()), fmt.Sprint(lastStoreFinish, store))
			}
			lastStoreFinish = store.Finish()
		}
		require.True(t, lastStoreFinish.Before(finished))

		// Check each writer wrote the expected number of times.
		for n := 0; n < nbWrites; n++ {
			v, exists := results[n]
			require.True(t, exists)
			require.Equal(t, uint64(nbWriters), v)
		}
	})
}

func BenchmarkStore(b *testing.B) {
	engine := metrics.NewEngine(logger)

	type structKeyType struct {
		n int
		s string
	}

	b.Run("non-concurrent insertion", func(b *testing.B) {
		b.Run("integer key type", func(b *testing.B) {
			b.Run("non existing keys", func(b *testing.B) {
				b.Run("using MetricsStore", func(b *testing.B) {
					store := engine.NewStore("id", time.Minute)
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						_ = store.Add(n, 1)
					}
				})

				b.Run("using sync.Map", func(b *testing.B) {
					var store sync.Map
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						store.Store(n, 1)
					}
				})
			})

			b.Run("already existing key", func(b *testing.B) {
				b.Run("using MetricsStore", func(b *testing.B) {
					store := engine.NewStore("id", time.Minute)
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						_ = store.Add(42, 1)
					}
				})

				b.Run("using sync.Map", func(b *testing.B) {
					var store sync.Map
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						store.Store(42, 1)
					}
				})
			})
		})

		b.Run("structure key type", func(b *testing.B) {
			b.Run("non existing keys", func(b *testing.B) {
				key := structKeyType{
					s: testlib.RandString(50),
				}

				b.Run("using MetricsStore", func(b *testing.B) {
					store := engine.NewStore("id", time.Minute)
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						key.n = n
						_ = store.Add(key, 1)
					}
				})

				b.Run("using sync.Map", func(b *testing.B) {
					var store sync.Map
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						key.n = n
						store.Store(key, 1)
					}
				})
			})

			b.Run("already existing key", func(b *testing.B) {
				key := structKeyType{
					n: 42,
					s: testlib.RandString(50),
				}
				b.Run("using MetricsStore", func(b *testing.B) {
					store := engine.NewStore("id", time.Minute)
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						_ = store.Add(key, 1)
					}
				})

				b.Run("using sync.Map", func(b *testing.B) {
					var store sync.Map
					b.ResetTimer()
					for n := 0; n < b.N; n++ {
						store.Store(key, 1)
					}
				})
			})
		})
	})

	b.Run("concurrent insertion", func(b *testing.B) {
		for p := 1; p <= 1000; p *= 10 {
			p := p
			b.Run(fmt.Sprintf("%d", p), func(b *testing.B) {
				b.Run("integer key type", func(b *testing.B) {
					b.Run("same non existing keys", func(b *testing.B) {
						b.Run("using MetricsStore", func(b *testing.B) {
							store := engine.NewStore("id", time.Minute)
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								n := 0
								for pb.Next() {
									_ = store.Add(n, 1)
									n++
								}
							})
						})

						b.Run("using sync.Map", func(b *testing.B) {
							var store sync.Map
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								n := 0
								for pb.Next() {
									store.Store(n, 1)
									n++
								}
							})
						})
					})

					b.Run("same key", func(b *testing.B) {
						b.Run("using MetricsStore", func(b *testing.B) {
							store := engine.NewStore("id", time.Minute)
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								for pb.Next() {
									_ = store.Add(42, 1)
								}
							})
						})

						b.Run("using sync.Map", func(b *testing.B) {
							var store sync.Map
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								for pb.Next() {
									store.Store(42, 1)
								}
							})
						})
					})
				})
				b.Run("structure key type", func(b *testing.B) {
					b.Run("same non existing keys", func(b *testing.B) {
						b.Run("using MetricsStore", func(b *testing.B) {
							store := engine.NewStore("id", time.Minute)
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								key := structKeyType{
									s: testlib.RandString(50),
								}
								for pb.Next() {
									_ = store.Add(key, 1)
									key.n++
								}
							})
						})

						b.Run("using sync.Map", func(b *testing.B) {
							var store sync.Map
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								key := structKeyType{
									s: testlib.RandString(50),
								}
								for pb.Next() {
									store.Store(key, 1)
									key.n++
								}
							})
						})
					})

					b.Run("same key", func(b *testing.B) {
						key := structKeyType{
							s: testlib.RandString(50),
							n: 42,
						}

						b.Run("using MetricsStore", func(b *testing.B) {
							store := engine.NewStore("id", time.Minute)
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								for pb.Next() {
									_ = store.Add(key, 1)
								}
							})
						})

						b.Run("using sync.Map", func(b *testing.B) {
							var store sync.Map
							b.ResetTimer()
							b.SetParallelism(p)
							b.RunParallel(func(pb *testing.PB) {
								for pb.Next() {
									store.Store(key, 1)
								}
							})
						})
					})
				})
			})
		}
	})
}

func BenchmarkUsage(b *testing.B) {
	engine := metrics.NewEngine(logger)

	for p := 1; p <= 1000; p *= 10 {
		p := p
		b.Run(fmt.Sprintf("parallelism/%d", p), func(b *testing.B) {
			b.Run("constant cpu time", func(b *testing.B) {
				b.Run("reference without metrics", func(b *testing.B) {
					b.SetParallelism(p)
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							doConstantCPUProcessing(1)
						}
					})
				})

				b.Run("integer key type", func(b *testing.B) {
					b.Run("concurrent writes to the same key", func(b *testing.B) {
						b.SetParallelism(p)
						store := engine.NewStore("id", time.Minute)
						b.ResetTimer()
						b.RunParallel(func(pb *testing.PB) {
							for pb.Next() {
								_ = store.Add(418, 1)
								_ = doConstantCPUProcessing(1)
							}
						})
					})

					b.Run("concurrent writes to multiple keys", func(b *testing.B) {
						b.SetParallelism(p)
						store := engine.NewStore("id", time.Minute)
						b.ResetTimer()
						b.RunParallel(func(pb *testing.PB) {
							n := 0
							for pb.Next() {
								_ = store.Add(n, 1)
								_ = doConstantCPUProcessing(1)
								n++
							}
						})
					})
				})
			})
		})
	}
}

// go:noinline
func doConstantCPUProcessing(n int) (r int) {
	for i := 0; i < int(math.Pow(1000, float64(n))); i++ {
		r += useCPU(i)
	}
	return r
}

// go:noinline
func useCPU(i int) int {
	return i + 10 - 2*3
}

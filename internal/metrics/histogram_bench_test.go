// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package metrics_test

import (
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/stretchr/testify/require"
)

type timeHistogramBench struct {
	setup func(*testing.B, *metrics.TimeHistogram)
	perG  func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram)
}

func benchTimeHistogram(b *testing.B, bench timeHistogramBench) {
	store := metrics.NewTimeHistogram(time.Minute, 100000)

	if bench.setup != nil {
		bench.setup(b, store)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var i int64
	b.RunParallel(func(pb *testing.PB) {
		id := int(atomic.AddInt64(&i, 1) - 1)
		bench.perG(b, pb, id*b.N, store)
	})
}

type perfHistogramBench struct {
	setup func(*testing.B, *metrics.PerfHistogram)
	perG  func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram)
}

func benchPerfHistogram(b *testing.B, bench perfHistogramBench) {
	store, err := metrics.NewPerfHistogram(time.Minute, 1, 2, 100000)
	require.NoError(b, err)

	if bench.setup != nil {
		bench.setup(b, store)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var i int64
	b.RunParallel(func(pb *testing.PB) {
		id := int(atomic.AddInt64(&i, 1) - 1)
		bench.perG(b, pb, id*b.N, store)
	})
}

func BenchmarkTimeHistogram(b *testing.B) {
	b.Run("mostly key hits", func(b *testing.B) {
		const hits, misses = 1023, 1

		benchTimeHistogram(b, timeHistogramBench{
			setup: func(_ *testing.B, store *metrics.TimeHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(i, 1); err != nil {
						b.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(int64(i%(hits+misses)), 1); err != nil {
						b.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("mostly misses", func(b *testing.B) {
		const hits, misses = 1, 1023

		benchTimeHistogram(b, timeHistogramBench{
			setup: func(_ *testing.B, store *metrics.TimeHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(i, 1); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(i%(hits+misses), 1); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("balanced", func(b *testing.B) {
		const hits, misses = 128, 128

		benchTimeHistogram(b, timeHistogramBench{
			setup: func(b *testing.B, store *metrics.TimeHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(i, 1); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram) {
				for ; pb.Next(); i++ {
					c := i % (hits + misses)
					var k int
					if c < hits {
						k = c
					} else {
						k = i
					}
					if err := store.Add(k, 1); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("unique", func(b *testing.B) {
		benchTimeHistogram(b, timeHistogramBench{
			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(i, 1); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("collision", func(b *testing.B) {
		benchTimeHistogram(b, timeHistogramBench{
			setup: func(_ *testing.B, store *metrics.TimeHistogram) {
				for i := 0; i < 100; i++ {
					if err := store.Add(i, 1); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.TimeHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(i%100, 1); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})
}

func BenchmarkPerfHistogram(b *testing.B) {
	b.Run("mostly key hits", func(b *testing.B) {
		const hits, misses = 1023, 1

		benchPerfHistogram(b, perfHistogramBench{
			setup: func(_ *testing.B, store *metrics.PerfHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(float64(i)); err != nil {
						b.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(float64(i % (hits + misses))); err != nil {
						b.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("mostly misses", func(b *testing.B) {
		const hits, misses = 1, 1023

		benchPerfHistogram(b, perfHistogramBench{
			setup: func(_ *testing.B, store *metrics.PerfHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(float64(i)); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(float64(i % (hits + misses))); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("balanced", func(b *testing.B) {
		const hits, misses = 128, 128

		benchPerfHistogram(b, perfHistogramBench{
			setup: func(b *testing.B, store *metrics.PerfHistogram) {
				for i := 0; i < hits; i++ {
					if err := store.Add(float64(i)); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram) {
				for ; pb.Next(); i++ {
					j := i % (hits + misses)
					var v float64
					if j < hits {
						v = float64(j)
					} else {
						v = float64(i)
					}
					if err := store.Add(float64(v)); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("unique", func(b *testing.B) {
		benchPerfHistogram(b, perfHistogramBench{
			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(float64(i)); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})

	b.Run("collision", func(b *testing.B) {
		benchPerfHistogram(b, perfHistogramBench{
			setup: func(_ *testing.B, store *metrics.PerfHistogram) {
				for i := 0; i < 100; i++ {
					if err := store.Add(float64(i)); err != nil {
						log.Fatal(err)
					}
				}
			},

			perG: func(b *testing.B, pb *testing.PB, i int, store *metrics.PerfHistogram) {
				for ; pb.Next(); i++ {
					if err := store.Add(float64(i % 100)); err != nil {
						log.Fatal(err)
					}
				}
			},
		})
	})
}

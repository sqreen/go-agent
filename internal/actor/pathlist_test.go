// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor_test

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestPathList(t *testing.T) {
	t.Run("concurrent accesses", func(t *testing.T) {
		paths := RandPathList(1000)
		tree := actor.NewPathListStore(paths)

		for i := range paths {
			path := paths[i] // new scope var
			t.Run("", func(t *testing.T) {
				t.Parallel()
				require.True(t, tree.Find(path))
			})
		}
	})
}

func BenchmarkPathList(b *testing.B) {
	b.Run("Lookup", func(b *testing.B) {
		b.Run("Random strings", func(b *testing.B) {
			for n := 1; n <= 1000000; n *= 10 {
				n := n
				paths := RandPathList(n)
				tree := actor.NewPathListStore(paths)
				b.Run(fmt.Sprintf("%d", len(paths)), func(b *testing.B) {
					b.ReportAllocs()
					for n := 0; n < b.N; n++ {
						// Pick a random path that was inserted
						ix := int(rand.Int63n(int64(len(paths))))
						path := paths[ix]
						found := tree.Find(path)
						if !found {
							// It should not happen
							b.FailNow()
						}
					}
				})
			}
		})
	})

	b.Run("Insertion", func(b *testing.B) {
		b.Run("Random strings", func(b *testing.B) {
			for n := 1; n <= 1000000; n *= 10 {
				n := n
				paths := RandPathList(n)
				b.Run(fmt.Sprintf("%d", len(paths)), func(b *testing.B) {
					b.ReportAllocs()
					for n := 0; n < b.N; n++ {
						actor.NewPathListStore(paths)
					}
				})
			}
		})
	})
}

func RandPathList(n int) (paths []string) {
	paths = make([]string, 0, n)

	for i := 0; i < n; i++ {
		// Insert unique rand values to strictly have n values
		for {
			path := testlib.RandUTF8String(10)
			pos := sort.SearchStrings(paths, path)

			if pos < len(paths) && paths[pos] == path {
				// path is already present, try another rand value
				continue
			}

			// path is not present and pos is the index where it should be inserted
			pathsBefore := paths[:pos]
			pathsAfter := paths[pos:]
			paths = append(pathsBefore, path)
			paths = append(paths, pathsAfter...)
			break
		}
	}

	return paths
}

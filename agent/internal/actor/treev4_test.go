package actor_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint8_tree"
	"github.com/stretchr/testify/require"
)

// Check the radix tree package does what we expect.
func TestRadixTreeAPI(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		tree := uint8_tree.NewTreeV4()

		ip, _, err := patricia.ParseIPFromString("1.2.3.5")
		require.NoError(t, err)
		increased, nbTags, err := tree.Add(*ip, 45, func(existing uint8, new uint8) bool {
			// Should not be called as this is the first tag
			t.FailNow()
			return true
		})
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		increased, nbTags, err = tree.Add(*ip, 46, func(existing uint8, new uint8) bool {
			// Should be called as this is no longer the first tag
			require.Equal(t, uint8(46), new)
			// True = it matches so do not add
			return true
		})
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.False(t, increased)

		// Find the tag for the IP, it should be the first one (45).
		tags, err := tree.FindTags(*ip)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(45))
	})

	t.Run("subsequent IPs with same tag values", func(t *testing.T) {
		tree := uint8_tree.NewTreeV4()

		ip1, _, err := patricia.ParseIPFromString("1.2.3.5")
		require.NoError(t, err)
		increased, nbTags, err := tree.Add(*ip1, 45, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		ip2, _, err := patricia.ParseIPFromString("1.2.3.4")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip2, 44, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		ip3, _, err := patricia.ParseIPFromString("1.2.3.6")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip3, 46, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		tags, err := tree.FindTags(*ip1)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(45))

		tags, err = tree.FindTags(*ip3)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(46))

		tags, err = tree.FindTags(*ip2)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(44))

		tags, err = tree.FindTagsWithFilter(*ip2, func(tag uint8) bool {
			require.Equal(t, tag, uint8(44))
			return tag == 44
		})
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(44))

		// Check that FindTags returns an array ordered by prefix-length
		net1, _, err := patricia.ParseIPFromString("10.1.2.0/24")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net1, 47, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		net2, _, err := patricia.ParseIPFromString("10.1.0.0/16")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net2, 48, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		net3, _, err := patricia.ParseIPFromString("10.0.0.0/8")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net3, 49, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		ip4, _, err := patricia.ParseIPFromString("10.1.2.33/32")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip4, 50, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		tags, err = tree.FindTags(*ip4)
		require.NoError(t, err)
		require.Equal(t, len(tags), 4)
		// Check that tags are returned in ordered.
		require.Equal(t, uint8(49), tags[0])
		require.Equal(t, uint8(48), tags[1])
		require.Equal(t, uint8(47), tags[2])
		require.Equal(t, uint8(50), tags[3])
	})

	t.Run("subsequent IPs with different tag values", func(t *testing.T) {
		tree := uint8_tree.NewTreeV4()

		ip1, _, err := patricia.ParseIPFromString("1.2.3.5")
		require.NoError(t, err)
		increased, nbTags, err := tree.Add(*ip1, 42, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		ip2, _, err := patricia.ParseIPFromString("1.2.3.4")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip2, 43, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		ip3, _, err := patricia.ParseIPFromString("1.2.3.6")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip3, 44, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		tags, err := tree.FindTags(*ip1)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(42))

		tags, err = tree.FindTags(*ip3)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(44))

		tags, err = tree.FindTags(*ip2)
		require.NoError(t, err)
		require.Equal(t, len(tags), 1)
		require.Equal(t, tags[0], uint8(43))
	})
}

func BenchmarkTree(b *testing.B) {
	b.Run("IPv4", func(b *testing.B) {
		b.Run("Lookup", func(b *testing.B) {
			b.Run("Random addresses", func(b *testing.B) {
				for n := 1; n <= 1000000; n *= 10 {
					n := n
					tree, cidrs := RandTreeV4(b, n, RandPatriciaIPv4Address)
					b.Run(fmt.Sprintf("%d", len(cidrs)), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							// Pick a random CIDR that was inserted
							ix := int(rand.Int63n(int64(len(cidrs))))
							cidr := cidrs[ix]
							found, _, err := tree.FindDeepestTag(cidr)
							if err != nil || !found {
								b.FailNow()
							}
						}
					})
				}
			})

			b.Run("Realisitic Random Networks", func(b *testing.B) {
				for n := 1; n <= 1000000; n *= 10 {
					n := n
					tree, cidrs := RandTreeV4(b, n, RandRealisticPatriciaCIDRv4)
					b.Run(fmt.Sprintf("%d", len(cidrs)), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							// Pick a random CIDR that was inserted
							ix := int(rand.Int63n(int64(len(cidrs))))
							cidr := cidrs[ix]
							found, _, err := tree.FindDeepestTag(cidr)
							if err != nil || !found {
								b.FailNow()
							}
						}
					})
				}
			})

			b.Run("Random Networks", func(b *testing.B) {
				for n := 1; n <= 1000000; n *= 10 {
					n := n
					tree, cidrs := RandTreeV4(b, n, RandPatriciaCIDRv4)
					b.Run(fmt.Sprintf("%d", len(cidrs)), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							// Pick a random CIDR that was inserted
							ix := int(rand.Int63n(int64(len(cidrs))))
							cidr := cidrs[ix]
							found, _, err := tree.FindDeepestTag(cidr)
							if err != nil || !found {
								b.FailNow()
							}
						}
					})
				}
			})
		})

		b.Run("Insertion", func(b *testing.B) {
			cidr, _, err := patricia.ParseIPFromString("1.2.3.4")
			require.NoError(b, err)

			b.Run("Consequitive addresses", func(b *testing.B) {
				b.ReportAllocs()
				tree := uint8_tree.NewTreeV4()
				for n := 0; n < b.N; n++ {
					cidr.Address += 1
					tree.Set(*cidr, 0)
				}
			})

			b.Run("Random addresses", func(b *testing.B) {
				b.ReportAllocs()
				tree := uint8_tree.NewTreeV4()
				for n := 0; n < b.N; n++ {
					cidr.Address = rand.Uint32()
					tree.Set(*cidr, 0)
				}
			})

			b.Run("Random networks", func(b *testing.B) {
				b.ReportAllocs()
				tree := uint8_tree.NewTreeV4()
				for n := 0; n < b.N; n++ {
					cidr.Address = rand.Uint32()
					cidr.Length = 1 + (uint(rand.Uint32()) % uint(32))
					tree.Set(*cidr, 0)
				}
			})
		})

		b.Run("Size", func(b *testing.B) {
			cidr := patricia.NewIPv4Address(rand.Uint32(), 32)

			b.Run("Consequitive addresses", func(b *testing.B) {
				for size := 1; size <= 1000000; size *= 10 {
					size := size
					b.Run(fmt.Sprint(size), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							RandTreeV4_ForBenchmark(b, size, func() patricia.IPv4Address {
								cidr.Address += 1
								return cidr
							})
						}
					})
				}
			})

			b.Run("Random addresses", func(b *testing.B) {
				for size := 1; size <= 1000000; size *= 10 {
					size := size
					b.Run(fmt.Sprint(size), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							RandTreeV4_ForBenchmark(b, size, func() patricia.IPv4Address {
								cidr.Address = rand.Uint32()
								return cidr
							})
						}
					})
				}
			})

			b.Run("Random networks", func(b *testing.B) {
				for size := 1; size <= 1000000; size *= 10 {
					size := size
					b.Run(fmt.Sprint(size), func(b *testing.B) {
						b.ReportAllocs()
						for n := 0; n < b.N; n++ {
							RandTreeV4_ForBenchmark(b, size, func() patricia.IPv4Address {
								cidr.Address = rand.Uint32()
								cidr.Length = 1 + (uint(rand.Uint32()) % uint(32))
								return cidr
							})
						}
					})
				}
			})

		})
	})
}

func RandTreeV4_ForBenchmark(t testing.TB, n int, randCIDRv4 func() patricia.IPv4Address) *uint8_tree.TreeV4 {
	tree := uint8_tree.NewTreeV4()

	for i := 0; i < n; i++ {
		for {
			cidr := randCIDRv4()
			added, _, _ := tree.Add(cidr, uint8(rand.Uint32()), func(payload, val uint8) bool {
				return payload == val
			})
			if added {
				break
			}
		}
	}

	return tree
}

func RandTreeV4(t testing.TB, n int, randCIDRv4 func() patricia.IPv4Address) (*uint8_tree.TreeV4, []patricia.IPv4Address) {
	tree := uint8_tree.NewTreeV4()
	cidrs := make([]patricia.IPv4Address, 0, n)

	for i := 0; i < n; i++ {
		for {
			cidr := randCIDRv4()
			added, _, _ := tree.Add(cidr, uint8(rand.Uint32()), func(payload, val uint8) bool {
				return payload == val
			})
			if added {
				cidrs = append(cidrs, cidr)
				break
			}
		}
	}

	return tree, cidrs
}

func RandPatriciaIPv4Address() patricia.IPv4Address {
	return patricia.NewIPv4Address(rand.Uint32(), 32)
}

func RandRealisticPatriciaCIDRv4() patricia.IPv4Address {
	return patricia.NewIPv4Address(rand.Uint32(), uint(10+(rand.Uint32()%23)))
}

func RandPatriciaCIDRv4() patricia.IPv4Address {
	return patricia.NewIPv4Address(rand.Uint32(), uint(1+(rand.Uint32()%32)))
}

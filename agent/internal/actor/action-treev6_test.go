// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor_test

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint8_tree"
	"github.com/stretchr/testify/require"
)

// Check the radix tree package does what we expect.
func TestRadixTreeV6API(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		tree := uint8_tree.NewTreeV6()

		_, ip, err := patricia.ParseIPFromString("1:2:3:4:5:6:7:8")
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

	t.Run("subsequent CIDR IPs", func(t *testing.T) {
		tree := uint8_tree.NewTreeV6()

		_, ip1, err := patricia.ParseIPFromString("1:2:3:4:5:6:7:8")
		require.NoError(t, err)
		increased, nbTags, err := tree.Add(*ip1, 45, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		_, ip2, err := patricia.ParseIPFromString("1:2:3:4:5:6:7:7")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*ip2, 44, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		_, ip3, err := patricia.ParseIPFromString("1:2:3:4:5:6:7:9")
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
		_, net1, err := patricia.ParseIPFromString("fd00::/24")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net1, 47, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		_, net2, err := patricia.ParseIPFromString("fd00::/16")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net2, 48, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		_, net3, err := patricia.ParseIPFromString("fd00::/8")
		require.NoError(t, err)
		increased, nbTags, err = tree.Add(*net3, 49, nil)
		require.NoError(t, err)
		require.Equal(t, nbTags, 1)
		require.True(t, increased)

		_, ip4, err := patricia.ParseIPFromString("fd00::42/128")
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
}

func BenchmarkTreeV6(b *testing.B) {
	b.Run("Lookup", func(b *testing.B) {
		b.Run("Random addresses", func(b *testing.B) {
			for n := 1; n <= 1000000; n *= 10 {
				n := n
				tree, cidrs := RandTreeV6(b, n, RandPatriciaIPv6Address)
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
				tree, cidrs := RandTreeV6(b, n, RandPatriciaCIDRv6)
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
		_, cidr, err := patricia.ParseIPFromString("1:2:3:4:5:6:7:8")
		require.NoError(b, err)

		b.Run("Consequitive addresses", func(b *testing.B) {
			b.ReportAllocs()
			tree := uint8_tree.NewTreeV6()
			for n := 0; n < b.N; n++ {
				cidr.Right += 1
				tree.Set(*cidr, 0)
			}
		})

		b.Run("Random addresses", func(b *testing.B) {
			b.ReportAllocs()
			tree := uint8_tree.NewTreeV6()
			for n := 0; n < b.N; n++ {
				cidr.Left = rand.Uint64()
				cidr.Right = rand.Uint64()
				tree.Set(*cidr, 0)
			}
		})

		b.Run("Random networks", func(b *testing.B) {
			b.ReportAllocs()
			tree := uint8_tree.NewTreeV6()
			for n := 0; n < b.N; n++ {
				cidr.Left = rand.Uint64()
				cidr.Right = rand.Uint64()
				cidr.Length = 1 + (uint(rand.Uint32()) % uint(8*net.IPv6len))
				tree.Set(*cidr, 0)
			}
		})
	})

	b.Run("Size", func(b *testing.B) {
		cidr := RandPatriciaIPv6Address()

		b.Run("Consequitive addresses", func(b *testing.B) {
			for size := 1; size <= 1000000; size *= 10 {
				size := size
				b.Run(fmt.Sprint(size), func(b *testing.B) {
					b.ReportAllocs()
					for n := 0; n < b.N; n++ {
						RandTreeV6_ForBenchmark(b, size, func() patricia.IPv6Address {
							cidr.Right += 1
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
						RandTreeV6_ForBenchmark(b, size, func() patricia.IPv6Address {
							cidr.Left = rand.Uint64()
							cidr.Right = rand.Uint64()
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
						RandTreeV6_ForBenchmark(b, size, func() patricia.IPv6Address {
							cidr.Left = rand.Uint64()
							cidr.Right = rand.Uint64()
							cidr.Length = 1 + (uint(rand.Uint32()) % uint(32))
							return cidr
						})
					}
				})
			}
		})
	})
}

func RandTreeV6_ForBenchmark(t testing.TB, n int, randCIDRv6 func() patricia.IPv6Address) *uint8_tree.TreeV6 {
	tree := uint8_tree.NewTreeV6()

	for i := 0; i < n; i++ {
		for {
			cidr := randCIDRv6()
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

func RandTreeV6(t testing.TB, n int, randCIDRv6 func() patricia.IPv6Address) (*uint8_tree.TreeV6, []patricia.IPv6Address) {
	tree := uint8_tree.NewTreeV6()
	cidrs := make([]patricia.IPv6Address, 0, n)

	for i := 0; i < n; i++ {
		for {
			cidr := randCIDRv6()
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

func RandPatriciaIPv6Address() patricia.IPv6Address {
	ip := RandIPv6()
	return patricia.NewIPv6Address(ip, net.IPv6len*8)
}

func RandPatriciaCIDRv6() patricia.IPv6Address {
	ip := RandIPv6()
	bits := net.IPv6len * 8
	return patricia.NewIPv6Address(ip, uint(1+rand.Uint32()%uint32(bits)))
}

func RandIPv6() net.IP {
	ip := make([]byte, net.IPv6len)
	rand.Read(ip)
	return net.IP(ip)
}

/*
 * Copyright 2019 Sqreen. All Rights Reserved.
 * Please refer to our terms for more information:
 * https://www.sqreen.io/terms.html
 */

package actor_test

import (
	"fmt"
	"math/rand"
	"net"
	"testing"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/stretchr/testify/require"
)

func TestCIDRIPListStore(t *testing.T) {
	t.Run("Random addresses", func(t *testing.T) {
		var cidrs []string
		// 4000 because the race detector cannot work with more than 8192
		// goroutines - we have here two entries per loop = 8000 values here
		for i := 0; i < 4000; i++ {
			cidrs = append(cidrs, RandIPv4().String())
			cidrs = append(cidrs, RandIPv6().String())
		}

		iplist, err := actor.NewCIDRIPListStore(cidrs)
		require.NoError(t, err)

		for _, cidr := range cidrs {
			cidr := cidr // new scope
			t.Run("Find", func(t *testing.T) {
				t.Parallel()
				listed, matched, err := iplist.Find(net.ParseIP(cidr))
				require.NoError(t, err)
				require.True(t, listed)
				require.Equal(t, cidr, matched)
			})
		}
	})

	t.Run("With networks", func(t *testing.T) {
		matchesNetv4 := "1.2.3.4/27"
		matchesIPv4 := "5.6.7.8"
		matchesNetv6 := "1:2:3:4:5:6:7:8/120"
		matchesIPv6 := "33:2:3:4:5:6:7:8"
		iplist, err := actor.NewCIDRIPListStore([]string{
			matchesNetv4,
			matchesIPv4,
			matchesNetv6,
			matchesIPv6,
		})
		require.NoError(t, err)

		for _, tc := range []struct {
			test    net.IP
			matches string
		}{
			{
				test:    net.IP{1, 2, 3, 5},
				matches: matchesNetv4,
			},
			{
				test:    net.IP{1, 2, 3, 4},
				matches: matchesNetv4,
			},
			{
				test:    net.IP{1, 2, 3, 3},
				matches: matchesNetv4,
			},
			{
				test:    net.IP{1, 2, 3, 0},
				matches: matchesNetv4,
			},
			{
				test:    net.IP{1, 2, 3, 1},
				matches: matchesNetv4,
			},
			{
				test: net.IP{1, 2, 2, 255},
			},
			{
				test: net.IP{1, 2, 4, 0},
			},
			{
				test:    net.IP{5, 6, 7, 8},
				matches: matchesIPv4,
			},
			{
				test:    net.ParseIP("1:2:3:4:5:6:7:8"),
				matches: matchesNetv6,
			},
			{
				test:    net.ParseIP("1:2:3:4:5:6:7:00ff"),
				matches: matchesNetv6,
			},
			{
				test:    net.ParseIP("1:2:3:4:5:6:7:0"),
				matches: matchesNetv6,
			},
			{
				test:    net.ParseIP("1:2:3:4:5:6:7:8"),
				matches: matchesNetv6,
			},
			{
				test: net.ParseIP("1:2:3:4:5:6:6:0100"),
			},
			{
				test: net.ParseIP("1:2:3:4:5:6:8:0"),
			},
			{
				test:    net.ParseIP("33:2:3:4:5:6:7:8"),
				matches: matchesIPv6,
			},
			{
				test:    net.ParseIP("33:2:3:4:5:6:7:8"),
				matches: matchesIPv6,
			},
		} {
			t.Run(fmt.Sprintf("Find(%s)", tc.test), func(t *testing.T) {
				t.Parallel()
				listed, matched, err := iplist.Find(tc.test)
				require.NoError(t, err)
				require.Equal(t, tc.matches != "", listed)
				require.Equal(t, tc.matches, matched)
			})
		}
	})
}

func BenchmarkCIDRIPListStore(b *testing.B) {
	b.Run("Lookup", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n // new scope
			iplist, ips := NewRandCIDRIPList(b, n)
			b.Run(fmt.Sprint(n), func(b *testing.B) {
				b.ReportAllocs()
				for n := 0; n < b.N; n++ {
					// Pick a random cidr that was inserted
					ix := int(rand.Int63n(int64(len(ips))))
					ip := ips[ix]
					listed, _, err := iplist.Find(ip)
					if err != nil || !listed {
						b.FailNow()
					}
				}
			})
		}
	})

	b.Run("Insertion", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n // new scope
			_, ips := NewRandIPList(n)
			b.Run(fmt.Sprint(n), func(b *testing.B) {
				for n := 0; n < b.N; n++ {
					_, _ = actor.NewCIDRIPListStore(ips)
				}
			})
		}
	})

	b.Run("Memory Pressure", func(b *testing.B) {
		for n := 1; n <= 1000000; n *= 10 {
			n := n // new scope
			_, ips := NewRandIPList(n)
			b.Run(fmt.Sprint(n), func(b *testing.B) {
				b.ReportAllocs()
				for n := 0; n < b.N; n++ {
					_, _ = actor.NewCIDRIPListStore(ips)
				}
			})
		}
	})
}

func NewRandCIDRIPList(b *testing.B, n int) (iplist *actor.CIDRIPListStore, ips []net.IP) {
	ips, ipsStr := NewRandIPList(n)
	iplist, err := actor.NewCIDRIPListStore(ipsStr)
	require.NoError(b, err)
	return iplist, ips
}

func NewRandIPList(n int) (ips []net.IP, ipsStr []string) {
	for i := 0; i < n; i++ {
		ipv4 := RandIPv4()
		ipv6 := RandIPv6()
		ips = append(ips, ipv4, ipv6)
		ipsStr = append(ipsStr, ipv4.String(), ipv6.String())
	}
	return
}

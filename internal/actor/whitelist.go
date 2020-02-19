/*
 * Copyright 2019 Sqreen. All Rights Reserved.
 * Please refer to our terms for more information:
 * https://www.sqreen.io/terms.html
 */

package actor

import (
	"net"

	"github.com/kentik/patricia"
)

// CIDRWhitelistStore is the set of data-structures to store CIDR IPv6 and IPv4
// whitelists. Locking is avoided by not having concurrent insertions and
// lookups. Therefore, a second whitelistStore is created when a new whitelist
// is received, and only swapping the whitelistStore pointer needs to be
// thread-safe.
type CIDRWhitelistStore struct {
	treeV4 *WhitelistTreeV4
	treeV6 *WhitelistTreeV6
}

type WhitelistTreeV4 actionTreeV4
type WhitelistTreeV6 actionTreeV6

func NewCIDRWhitelistStore(cidrs []string) (*CIDRWhitelistStore, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}
	whitelistv4 := newActionTreeV4(maxStoreActions)
	whitelistv6 := newActionTreeV6(maxStoreActions)
	var hasIPv4, hasIPv6 bool // true when at least one IP was added to the tree
	for _, cidr := range cidrs {
		action := newWhitelistAction(cidr)
		ipv4, ipv6, err := patricia.ParseIPFromString(cidr)
		if err != nil {
			return nil, err
		}
		if ipv4 != nil {
			if err := whitelistv4.addAction(ipv4, action); err != nil {
				return nil, err
			}
			hasIPv4 = true
		} else if ipv6 != nil {
			if err := whitelistv6.addAction(ipv6, action); err != nil {
				return nil, err
			}
			hasIPv6 = true
		}
	}
	// Release empty whitelist trees when nothing was added to them.
	if !hasIPv4 {
		whitelistv4 = nil
	}
	if !hasIPv6 {
		whitelistv6 = nil
	}
	return &CIDRWhitelistStore{
		treeV4: (*WhitelistTreeV4)(whitelistv4),
		treeV6: (*WhitelistTreeV6)(whitelistv6),
	}, nil
}

func (s *CIDRWhitelistStore) Find(ip net.IP) (whitelisted bool, matched string, err error) {
	var action Action
	if stdIPv4 := ip.To4(); stdIPv4 != nil {
		tree := (*actionTreeV4)(s.treeV4)
		if tree == nil {
			return false, "", nil
		}
		IPv4 := patricia.NewIPv4AddressFromBytes(stdIPv4, ipv4Bits)
		action, err = tree.findAction(&IPv4)
	} else if stdIPv6 := ip.To16(); stdIPv6 != nil {
		// warning: the previous condition is also true with an ipv4 address (as
		// they can be represented using ipv6 ::ffff:ipv4), so testing the ipv4
		// first is important to avoid entering this case with ipv4 addresses.
		tree := (*actionTreeV6)(s.treeV6)
		if tree == nil {
			return false, "", nil
		}
		IPv6 := patricia.NewIPv6Address(stdIPv6, ipv6Bits)
		action, err = tree.findAction(&IPv6)
	}
	if err != nil {
		return false, "", err
	}
	if action == nil {
		return false, "", nil
	}
	return true, action.ActionID(), nil
}

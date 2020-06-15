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

type (
	// CIDRIPListStore is the set of data-structures to store CIDR IPv6 and IPv4
	// IPLists. Locking is avoided by not having concurrent insertions and
	// lookups. Therefore, a second CIDRIPListStore is created when a new IPList
	// is received, and only swapping the pointers needs to be thread-safe.
	CIDRIPListStore struct {
		treeV4 *IPListTreeV4
		treeV6 *IPListTreeV6
	}

	IPListTreeV4 actionTreeV4
	IPListTreeV6 actionTreeV6
)

func NewCIDRIPListStore(cidrs []string) (*CIDRIPListStore, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}
	IPListv4 := newActionTreeV4(maxStoreActions)
	IPListv6 := newActionTreeV6(maxStoreActions)
	var hasIPv4, hasIPv6 bool // true when at least one IP was added to the tree
	for _, cidr := range cidrs {
		action := newIPListAction(cidr)
		ipv4, ipv6, err := patricia.ParseIPFromString(cidr)
		if err != nil {
			return nil, err
		}
		if ipv4 != nil {
			if err := IPListv4.addAction(ipv4, action); err != nil {
				return nil, err
			}
			hasIPv4 = true
		} else if ipv6 != nil {
			if err := IPListv6.addAction(ipv6, action); err != nil {
				return nil, err
			}
			hasIPv6 = true
		}
	}
	// Release empty IPList trees when nothing was added to them.
	if !hasIPv4 {
		IPListv4 = nil
	}
	if !hasIPv6 {
		IPListv6 = nil
	}
	return &CIDRIPListStore{
		treeV4: (*IPListTreeV4)(IPListv4),
		treeV6: (*IPListTreeV6)(IPListv6),
	}, nil
}

func (s *CIDRIPListStore) Find(ip net.IP) (exists bool, matched string, err error) {
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

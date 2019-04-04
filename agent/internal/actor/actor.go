// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Actor store
//
// The actor store is a central agent store associating actor IP addresses and
// user identifiers to security actions provided by the backend. Since it is
// used in HTTP request handlers, it is designed to be as efficient as possible
// to avoid slowing down requests. An important design constraint is the fact
// that the sooner a request is handled, the sooner its memory is released
// (goroutines and memory they used). So time-efficiency is considered as a
// better general memory-efficiency here.
package actor

import (
	"net"
	"sync"

	"github.com/kentik/patricia"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/plog"
)

// Store is the structure associating IP addresses or user IDs to security
// actions such as whitelisting and blacklisting. It wraps several underlying
// memory- and cpu-efficient data-structures, and provides the API the agent
// expects. It is designed to have the shortest possible lookup time from HTTP
// request handlers while providing the ability the load other security actions
// concurrently, without locking the store operations. To do so, when a new set
// of actions is received, a new store is created while still using the current
// one, and only the access to the store pointer is synchronized using a
// reader/writer mutex (mutual-exclusion of readers and writers with 1 writer
// and N readers at a time). This operation is therefore limited to the time to
// modify the store pointer, hence the smallest possible locking time.
type Store struct {
	// Mutex for RW accesses to the store pointer. By design, a store can be
	// replaced while it is being used by other requests. It allows to have the
	// shortest concurrent access times to the store by avoiding blocking the
	// entire store methods.
	lock  sync.RWMutex
	store *store

	logger *plog.Logger
}

func NewStore(logger *plog.Logger) *Store {
	return &Store{
		logger: plog.NewLogger("actors", logger),
	}
}

// getStore is a thread-safe store pointer getter.
func (s *Store) getStore() (store *store) {
	s.lock.RLock()
	store = s.store
	s.lock.RUnlock()
	return
}

// setStore is a thread-safe store pointer setter.
func (s *Store) setStore(store *store) {
	s.lock.Lock()
	s.store = store
	s.lock.Unlock()
}

// FindIP returns the security action of the given IP v4/v6 address. The
// returned boolean `exists` is `false` when it is not present in the store,
// `true` otherwise.
func (s *Store) FindIP(ip net.IP) (action Action, exists bool, err error) {
	store := s.getStore()
	if store == nil {
		return nil, false, nil
	}

	if stdIPv4 := ip.To4(); stdIPv4 != nil {
		tree := store.treeV4
		if tree == nil {
			return nil, false, nil
		}
		IPv4 := patricia.NewIPv4AddressFromBytes(stdIPv4, ipv4Bits)
		action, err = tree.findAction(&IPv4)
	} else if stdIPv6 := ip.To16(); stdIPv6 != nil {
		// warning: the previous condition is also true with ipv4 address (as they
		// can be represented using ipv6 ::ffff:ipv4), so testing the ipv4 first is
		// important to avoid entering this case even with ipv6 addresses.
		tree := store.treeV6
		if tree == nil {
			return nil, false, nil
		}
		IPv6 := patricia.NewIPv6Address(stdIPv6, ipv6Bits)
		action, err = tree.findAction(&IPv6)
	}

	if err != nil {
		return nil, false, err
	}

	// action may be nil if ip does not exist in the tree.
	return action, action != nil, nil
}

// SetActions creates a new action store and then replaces the current one. The
// new store is being built while still performing store methods on the current
// one.
func (s *Store) SetActions(actions []api.ActionsPackResponse_Action) error {
	store, err := newStore(actions)
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.setStore(store)
	return nil
}

// store is the set of data-structures the actor store can use at run time.
// Locking in the data-structure methods is avoided by not having concurrent
// insertions and lookups, and therefore a second store can be created when a
// new one needs to be created. Only the store pointer swapping needs to be
// thread-safe.
type store struct {
	treeV4 *treeV4
	treeV6 *treeV6
}

func newStore(actions []api.ActionsPackResponse_Action) (*store, error) {
	if len(actions) == 0 {
		return nil, nil
	}

	store := new(store)

	for _, action := range actions {
		err := store.addAction(action)
		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *store) addAction(action api.ActionsPackResponse_Action) error {
	switch action.Action {
	case actionKind_BlockIP:
		var blockIP Action = newBlockIPAction(action.ActionId)
		if action.Duration > 0 {
			blockIP = withDuration(blockIP, action.Duration)
		}
		cidrs := action.Parameters.IpCidr
		if len(cidrs) == 0 {
			return errors.Errorf("could not add action `%s`: empty list of CIDRs", action.ActionId)
		}

		err := s.addCIDRList(cidrs, blockIP)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *store) addCIDRList(cidrs []string, action Action) error {
	for _, cidr := range cidrs {
		ip4, ip6, err := patricia.ParseIPFromString(cidr)
		if err != nil {
			return err
		}
		if ip4 != nil {
			if err := s.addCIDRv4(ip4, action); err != nil {
				return err
			}
		} else if ip6 != nil {
			if err := s.addCIDRv6(ip6, action); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *store) addCIDRv4(ip *patricia.IPv4Address, action Action) error {
	if s.treeV4 == nil {
		s.treeV4 = newTreeV4(maxStoreActions)
	}

	return s.treeV4.addAction(ip, action)
}

func (s *store) addCIDRv6(ip *patricia.IPv6Address, action Action) error {
	if s.treeV6 == nil {
		s.treeV6 = newTreeV6(maxStoreActions)
	}
	return s.treeV6.addAction(ip, action)
}

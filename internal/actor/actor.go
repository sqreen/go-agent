// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor

import (
	"crypto/sha256"
	"math"
	"net"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/kentik/patricia"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/plog"
)

// Store is the structure associating IP addresses or user IDs to security
// actions such as passlisting and denylisting. It wraps several underlying
// memory- and cpu-efficient data-structures, and provides the API the agent
// needs. It is designed to have the shortest possible lookup time from HTTP
// request handlers while providing the ability the load other security actions
// concurrently, without locking the actionStore operations. To do so, when a
// new set of actions is received, a new actionStore is created while still
// using the current one, and only the access to the actionStore pointer is
// synchronized using a reader/writer mutex (mutual-exclusion of readers and
// writers with 1 writer and N readers at a time). This operation is therefore
// limited to the time to modify the actionStore pointer, hence the smallest
// possible locking time.
type Store struct {
	// The store of security response actions per IP address or user identifier.
	actionStore *actionStore
	// The store of the IP passlist.
	cidrIPPasslistStore *CIDRIPListStore
	// The store of the path passlist.
	pathPasslistStore *PathListStore

	logger *plog.Logger
}

func NewStore(logger *plog.Logger) *Store {
	return &Store{
		logger: logger,
	}
}

// SetPathPasslist creates a new passlist store and then replaces the current
// one. The new store is built while allowing accesses to the current one.
func (s *Store) SetPathPasslist(paths []string) {
	store := NewPathListStore(paths)
	s.setPathPasslistStore(store)
}

// getCIDRPasslistStore is a thread-safe store getter.
func (s *Store) getPathPasslistStore() (store *PathListStore) {
	return (*PathListStore)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.pathPasslistStore))))
}

// setPathPasslistStore is a thread-safe store setter.
func (s *Store) setPathPasslistStore(store *PathListStore) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.pathPasslistStore)), unsafe.Pointer(store))
}

// IsPathAllowed returns true when the given path matched a passlist entry.
// The error is non-nil when an internal error occurred.
func (s *Store) IsPathAllowed(path string) (allowed bool) {
	passlist := s.getPathPasslistStore()
	if passlist == nil {
		return false
	}
	return passlist.Find(path)
}

// SetCIDRIPPasslist creates a new passlist store and then replaces the current
// one. The new store is built while allowing accesses to the current one.
func (s *Store) SetCIDRIPPasslist(cidrs []string) error {
	store, err := NewCIDRIPListStore(cidrs)
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.setCIDRPasslistStore(store)
	return nil
}

// getCIDRPasslistStore is a thread-safe store getter.
func (s *Store) getCIDRPasslistStore() (store *CIDRIPListStore) {
	return (*CIDRIPListStore)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.cidrIPPasslistStore))))
}

// setCIDRPasslistStore is a thread-safe store setter.
func (s *Store) setCIDRPasslistStore(store *CIDRIPListStore) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.cidrIPPasslistStore)), unsafe.Pointer(store))
}

// IsIPAllowed returns true when the given IP address matched a passlist
// entry. This matched passlist entry is also returned. The error is non-nil
// when an internal error occurred.
func (s *Store) IsIPAllowed(ip net.IP) (allowed bool, matchedCIDR string, err error) {
	passlist := s.getCIDRPasslistStore()
	if passlist == nil {
		return false, "", nil
	}
	return passlist.Find(ip)
}

// getActionStore is a thread-safe actionStore pointer getter.
func (s *Store) getActionStore() (store *actionStore) {
	return (*actionStore)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.actionStore))))
}

// setActionStore is a thread-safe actionStore pointer setter.
func (s *Store) setActionStore(store *actionStore) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&s.actionStore)), unsafe.Pointer(store))
}

// FindIP returns the security action of the given IP v4/v6 address. The
// returned boolean `exists` is `false` when it is not present in the
// actionStore, `true` otherwise.
func (s *Store) FindIP(ip net.IP) (action Action, exists bool, err error) {
	store := s.getActionStore()
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
		// important to avoid entering this case with ipv4 addresses.
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

// FindUser returns the security action of the given userID map. The returned
// boolean `exists` is `false` when it is not present in the actionStore, `true`
// otherwise.
func (s *Store) FindUser(userID map[string]string) (action Action, exists bool) {
	store := s.getActionStore()
	if store == nil {
		return nil, false
	}
	users := store.users
	if len(users) == 0 {
		return nil, false
	}
	hash := NewUserIdentifiersHash(userID)
	action, exists = users[hash]

	// Check if the action is timed.
	if timed, implementsTimed := action.(Timed); implementsTimed && timed.Expired() {
		// The action is not removed from the map to avoid locking it with a
		// RWMutex.
		return nil, false
	}

	return
}

// SetActions creates a new action store and then replaces the current one. The
// new store is built while allowing accesses to the current one.
func (s *Store) SetActions(actions []api.ActionsPackResponse_Action) error {
	store, err := newActionStore(actions)
	if err != nil {
		s.logger.Error(err)
		return err
	}
	s.setActionStore(store)
	return nil
}

// actionStore is the set of data-structures the actor actionStore can use at
// run time. Locking in the data-structure methods is avoided by not having
// concurrent insertions and lookups, and therefore a second actionStore can be
// created when a new one needs to be created. Only the actionStore pointer
// swapping needs to be thread-safe.
type actionStore struct {
	treeV4 *actionTreeV4
	treeV6 *actionTreeV6
	users  userActionMap
}

type userActionMap map[UserIdentifiersHash]Action

func newActionStore(actions []api.ActionsPackResponse_Action) (*actionStore, error) {
	if len(actions) == 0 {
		return nil, nil
	}

	store := new(actionStore)

	for _, action := range actions {
		err := store.addAction(action)
		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *actionStore) addAction(action api.ActionsPackResponse_Action) (err error) {
	switch action.Action {
	case actionKindBlockIP:
		err = s.addBlockIPAction(action)
	case actionKindBlockUser:
		err = s.addBlockUserAction(action)
	case actionKindRedirectIP:
		err = s.addRedirectIPAction(action)
	case actionKindRedirectUser:
		err = s.addRedirectUserAction(action)
	}
	return err
}

func (s *actionStore) addBlockIPAction(action api.ActionsPackResponse_Action) error {
	duration, err := float64ToDuration(action.Duration)
	if err != nil {
		return err
	}
	var blockIP Action = newBlockAction(action.ActionId)
	if duration > 0 {
		blockIP = withDuration(blockIP, duration)
	}
	cidrs := action.Parameters.IpCidr
	if len(cidrs) == 0 {
		return errors.Errorf("could not add action `%s`: empty list of CIDRs", action.ActionId)
	}
	return s.addCIDRList(cidrs, blockIP)
}

func (s *actionStore) addRedirectIPAction(action api.ActionsPackResponse_Action) error {
	duration, err := float64ToDuration(action.Duration)
	if err != nil {
		return err
	}
	var redirectIP Action
	redirectIP, err = newRedirectAction(action.ActionId, action.Parameters.Url)
	if duration > 0 {
		redirectIP = withDuration(redirectIP, duration)
	}
	cidrs := action.Parameters.IpCidr
	if len(cidrs) == 0 {
		return errors.Errorf("could not add action `%s`: empty list of CIDRs", action.ActionId)
	}
	return s.addCIDRList(cidrs, redirectIP)
}

func (s *actionStore) addRedirectUserAction(action api.ActionsPackResponse_Action) error {
	duration, err := float64ToDuration(action.Duration)
	if err != nil {
		return err
	}
	var redirectUser Action
	redirectUser, err = newRedirectAction(action.ActionId, action.Parameters.Url)
	if duration > 0 {
		redirectUser = withDuration(redirectUser, duration)
	}
	users := action.Parameters.Users
	if len(users) == 0 {
		return errors.Errorf("could not add action `%s`: empty list of users", action.ActionId)
	}
	return s.addUserList(users, redirectUser)
}

// Convert a float64 to a `time.Duration` by making sure it doesn't overflow.
func float64ToDuration(duration float64) (time.Duration, error) {
	if duration <= math.MinInt64 || duration >= math.MaxInt64 {
		return 0, errors.Errorf("could not convert the time duration `%f` to seconds due to int64 overflow", duration)
	}
	return time.Duration(duration) * time.Second, nil
}

func (s *actionStore) addCIDRList(cidrs []string, action Action) error {
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

func (s *actionStore) addCIDRv4(ip *patricia.IPv4Address, action Action) error {
	if s.treeV4 == nil {
		s.treeV4 = newActionTreeV4(maxStoreActions)
	}
	return s.treeV4.addAction(ip, action)
}

func (s *actionStore) addCIDRv6(ip *patricia.IPv6Address, action Action) error {
	if s.treeV6 == nil {
		s.treeV6 = newActionTreeV6(maxStoreActions)
	}
	return s.treeV6.addAction(ip, action)
}

func (s *actionStore) addBlockUserAction(action api.ActionsPackResponse_Action) error {
	duration, err := float64ToDuration(action.Duration)
	if err != nil {
		return err
	}
	var blockUser Action = newBlockAction(action.ActionId)
	if duration > 0 {
		blockUser = withDuration(blockUser, duration)
	}
	users := action.Parameters.Users
	if len(users) == 0 {
		return errors.Errorf("could not add action `%s`: empty list of users", action.ActionId)
	}
	return s.addUserList(users, blockUser)
}

func (s *actionStore) addUserList(users []map[string]string, action Action) error {
	if len(s.users)+len(users) >= maxStoreActions {
		return errors.Errorf("number of actions `%d` exceeds `%d`", len(users), maxStoreActions)
	}

	if s.users == nil {
		s.users = make(userActionMap, len(users))
	}
	for _, user := range users {
		s.addUser(user, action)
	}
	return nil
}

func (s *actionStore) addUser(identifiers map[string]string, action Action) {
	hash := NewUserIdentifiersHash(identifiers)
	s.users[hash] = action
}

// UserIdentifiersHash is a type suitable to be used as key type of the map of
// user actions. It is therefore an array, as slices cannot be used as map key
// types.
type UserIdentifiersHash [sha256.Size]byte

func NewUserIdentifiersHash(id map[string]string) UserIdentifiersHash {
	var hash UserIdentifiersHash
	for k, v := range id {
		k := sha256.Sum256([]byte(k))
		v := sha256.Sum256([]byte(v))
		for i := 0; i < len(hash); i++ {
			hash[i] += k[i] + v[i]
		}
	}
	return hash
}

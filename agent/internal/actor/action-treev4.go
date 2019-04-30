// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor

import (
	"math"

	"github.com/pkg/errors"

	"github.com/kentik/patricia"
	"github.com/kentik/patricia/uint32_tree"
)

// IPv4 data-structure mapping CIDR IPv4 addresses to actions. The underlying
// radix-tree is used as an index to an array of actions. The number of actions
// is limited.
type actionTreeV4 struct {
	tree            *uint32_tree.TreeV4
	actions         []Action
	maxStoreActions int
}

func newActionTreeV4(maxStoreActions int) *actionTreeV4 {
	return &actionTreeV4{
		tree:            uint32_tree.NewTreeV4(),
		maxStoreActions: maxStoreActions,
	}
}

// addAction adds an action for the given CIDR IPv4. Only one action is stored
// per CIDR IPv4.
func (t *actionTreeV4) addAction(ip *patricia.IPv4Address, action Action) error {
	if len(t.actions) >= math.MaxUint32 {
		return errors.Errorf("too many actions: the number of actions exceeds the maximum index value")
	}
	if len(t.actions) >= t.maxStoreActions {
		return errors.Errorf("too many actions: the number of actions `%d` exceeds `%d`", len(t.actions), t.maxStoreActions)
	}

	// Assume the CIDR IPv4 is not already in the tree by taking a new action
	// index in the array
	tag := len(t.actions)
	// Try to add it thanks to a special match-function that is only called when a
	// tag already exists. If it does, return true and reuse the existing tag.
	added, _, err := t.tree.Add(*ip, uint32(tag), func(current uint32, _ uint32) bool {
		// Called only when not already existing in the tree. Reuse the existing
		// current tag and overwrite the action.
		t.actions[current] = action
		// Return that it already exists to avoid adding it.
		return true
	})
	// When added is true, it means the tag was added, so we need to append the
	// new action in the array at the new tag index.
	if added {
		t.actions = append(t.actions, action)
	}
	return err
}

// findAction returns the most specific (deepest in the tree) security action
// associated to a given CIDR IPv4 `ip`. It is nil when it does not exist or if
// it has expired.
func (t *actionTreeV4) findAction(ip *patricia.IPv4Address) (Action, error) {
	tags, err := t.tree.FindTagsWithFilter(*ip, actionsNotExpiredFilter(t.actions))
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	// Returned tags are ordered by matching prefix length, ie. the right-most is
	// the deepest match (eg. match in a /16, match in a /24, and match in a /32).
	tag := tags[len(tags)-1]
	return t.actions[tag], nil
}

// actionsNotExpiredFilter returns true when the given action is not expired or
// doesn't implement the `Timed` interface.
func actionsNotExpiredFilter(actions []Action) func(i uint32) bool {
	return func(i uint32) bool {
		action := actions[i]
		timed, implementsTimed := action.(Timed)
		if !implementsTimed {
			return true
		}
		return !timed.Expired()
	}
}

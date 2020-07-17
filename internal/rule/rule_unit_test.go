// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"io"
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

type hookMockup struct{}

func (h hookMockup) Attach(...sqhook.PrologCallback) error {
	panic("should not be called")
	// TODO: better API to avoid that? the map only needs a "comparable" key and
	//  doesn't matter about the hook interface.
}

func TestHookDescriptors(t *testing.T) {
	// Not actual callbacks but enough for this unit test.
	// We need to use distinct types to correctly check the ordering.

	t.Run("multiple callbacks having the same priority", func(t *testing.T) {
		var m = hookDescriptorMap{}
		key := hookMockup{}
		m.Add(key, 1, 1)
		m.Add(key, 2, 1)
		m.Add(key, 3, 1)
		m.Add(key, 4, 1)
		d := m[key]
		require.Equal(t, []int{1, 1, 1, 1}, d.priorities)
		require.Equal(t, []sqhook.PrologCallback{1, 2, 3, 4}, d.callbacks)
		require.Nil(t, d.closers)
	})

	t.Run("multiple callbacks having distinct priorities", func(t *testing.T) {
		var m = hookDescriptorMap{}
		key := hookMockup{}

		m.Add(key, 3, 2)
		m.Add(key, 5, 3)
		m.Add(key, 4, 2)
		m.Add(key, 1, 1)
		m.Add(key, 6, 3)
		m.Add(key, 2, 1)
		d := m[key]
		require.Equal(t, []int{1, 1, 2, 2, 3, 3}, d.priorities)
		require.Equal(t, []sqhook.PrologCallback{1, 2, 3, 4, 5, 6}, d.callbacks)
		require.Nil(t, d.closers)
	})

	t.Run("multiple callbacks with close methods", func(t *testing.T) {
		var m = hookDescriptorMap{}
		key := hookMockup{}
		m.Add(key, myFakeCallback(7), 10)
		m.Add(key, 3, 2)
		m.Add(key, myFakeCallback(1), 1)
		m.Add(key, 2, 1)
		m.Add(key, myFakeCallback(5), 3)
		m.Add(key, 4, 2)
		m.Add(key, 6, 3)

		d := m[key]
		require.Equal(t, []int{1, 1, 2, 2, 3, 3, 10}, d.priorities)
		require.Equal(t, []sqhook.PrologCallback{myFakeCallback(1), 2, 3, 4, myFakeCallback(5), 6, myFakeCallback(7)}, d.callbacks)
		require.Equal(t, []io.Closer{myFakeCallback(7), myFakeCallback(1), myFakeCallback(5)}, d.closers)
	})
}

type myFakeCallback int

func (m myFakeCallback) Close() error { return nil }

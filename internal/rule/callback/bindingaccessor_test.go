// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/stretchr/testify/require"
)

func TestBindingAccessorLibrary(t *testing.T) {
	lib := callback.NewLibraryBindingAccessorContext()

	t.Run("Array Library", func(t *testing.T) {
		t.Run("Prepend", func(t *testing.T) {
			t.Run("", func(t *testing.T) {
				got, err := lib.Array.Prepend([]string{"b", "c", "d"}, "a")
				require.NoError(t, err)
				require.Equal(t, []string{"a", "b", "c", "d"}, got)
			})

			t.Run("", func(t *testing.T) {
				got, err := lib.Array.Prepend([]int{}, 1)
				require.NoError(t, err)
				require.Equal(t, []int{1}, got)
			})

			t.Run("type mismatch", func(t *testing.T) {
				require.Panics(t, func() {
					lib.Array.Prepend([]int{}, "a")
				})
			})

			t.Run("nil slice value", func(t *testing.T) {
				got, err := lib.Array.Prepend(([]int)(nil), 1)
				require.NoError(t, err)
				require.Equal(t, []int{1}, got)
			})
		})
	})
}

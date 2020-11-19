// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqerrors_test

import (
	"errors"
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/stretchr/testify/require"
)

func TestWithInfo(t *testing.T) {
	t.Run("single info", func(t *testing.T) {
		err := errors.New("an error")
		info := map[string]string{
			"k1": "v1",
			"k2": "v2",
		}
		err = sqerrors.WithInfo(err, info)
		err = sqerrors.Wrap(err, "an error occurred")
		got := sqerrors.Info(err)
		require.Equal(t, info, got)
	})

	t.Run("multiple info", func(t *testing.T) {
		err := errors.New("an error")
		err = sqerrors.WithInfo(err, map[string]string{
			"k1": "v1",
			"k2": "v2",
		})
		err = sqerrors.Wrap(err, "an error occurred")
		err = sqerrors.WithInfo(err, map[string]string{"key": "value"})
		err = sqerrors.Wrap(err, "an error occurred")
		err = sqerrors.Wrap(err, "an error occurred")
		err = sqerrors.WithInfo(err, "what ever")
		err = sqerrors.Wrap(err, "an error occurred")
		err = sqerrors.Wrap(err, "an error occurred")
		err = sqerrors.WithInfo(err, 33)

		// Check that we get the earliest level
		got := sqerrors.Info(err)
		require.Equal(t, 33, got)
	})
}

func TestWithKey(t *testing.T) {
	t.Run("usage", func(t *testing.T) {
		// Checking the Go assumption that the type is correctly taken into account
		type (
			t1 struct{}
			t2 struct{}
		)
		require.NotEqual(t, t1{}, t2{})

		err := errors.New("an error")
		key := t1{}
		err = sqerrors.WithKey(err, key)
		err = sqerrors.Wrap(err, "an error occurred")
		got, ok := sqerrors.Key(err)
		require.True(t, ok)
		require.Equal(t, key, got)

		// Test two defined types identical but in distinct code blocks indeed two
		// distinct types
		err1 := sqerrors.New("my error")
		{
			type t3 struct{}
			err1 = sqerrors.WithKey(err1, t3{})
		}
		err2 := sqerrors.New("my error")
		{
			type t3 struct{}
			err2 = sqerrors.WithKey(err2, t3{})
		}
		k1, exists := sqerrors.Key(err1)
		require.True(t, exists)
		k2, exists := sqerrors.Key(err2)
		require.True(t, exists)
		require.NotEqual(t, k1, k2)
	})

	t.Run("nested keys", func(t *testing.T) {
		err := errors.New("an error")
		err = sqerrors.WithKey(err, "k1")
		err = sqerrors.WithKey(err, "k2")
		err = sqerrors.WithKey(err, "k3")
		key, exists := sqerrors.Key(err)
		require.True(t, exists)
		require.Equal(t, "k1", key)
	})

	t.Run("no key", func(t *testing.T) {
		err := errors.New("an error")
		key, exists := sqerrors.Key(err)
		require.False(t, exists)
		require.Nil(t, key)
	})
}

func TestErrorCollection(t *testing.T) {
	var errs sqerrors.ErrorCollection
	errs.Add(errors.New("error 1"))
	errs.Add(errors.New("error 2"))
	errs.Add(errors.New("error 3"))
	errs.Add(errors.New("error 4"))
	require.Equal(t, "multiple errors occurred: (error 1) error 1; (error 2) error 2; (error 3) error 3; (error 4) error 4", errs.Error())
}

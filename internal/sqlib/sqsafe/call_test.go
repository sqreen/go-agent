// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsafe_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

func TestCall(t *testing.T) {
	t.Run("without error", func(t *testing.T) {
		err := sqsafe.Call(func() error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("with a regular error", func(t *testing.T) {
		err := sqsafe.Call(func() error {
			return xerrors.New("oops")
		})
		require.Error(t, err)
		require.Equal(t, "oops", err.Error())
	})

	t.Run("with a panic string error", func(t *testing.T) {
		err := sqsafe.Call(func() error {
			panic("oops")
			return nil
		})
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
		require.Equal(t, "oops", panicErr.Err.Error())
	})

	t.Run("with a panic error", func(t *testing.T) {
		origErr := xerrors.New("oops")
		err := sqsafe.Call(func() error {
			panic(origErr)
			return nil
		})
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
	})

	t.Run("with another panic argument type", func(t *testing.T) {
		err := sqsafe.Call(func() error {
			panic(33.7)
			return nil
		})
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
		require.Equal(t, "33.7", panicErr.Err.Error())
	})

	t.Run("with a nil panic argument value", func(t *testing.T) {
		err := sqsafe.Call(func() error {
			// This case cannot be differentiated yet.
			panic(nil)
			return xerrors.New("oops")
		})
		require.NoError(t, err)
	})
}

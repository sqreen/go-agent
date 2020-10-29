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

func TestGo(t *testing.T) {
	t.Run("without error", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			return nil
		}, ch)
		err := <-ch
		require.NoError(t, err)
	})

	t.Run("with a regular error", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			return xerrors.New("oops")
		}, ch)
		err := <-ch
		require.Error(t, err)
		require.Equal(t, "oops", err.Error())
	})

	t.Run("with a panic string error", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			panic("oops")
			return nil
		}, ch)
		err := <-ch
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
		require.Equal(t, "oops", panicErr.Err.Error())
	})

	t.Run("with a panic error", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			panic(xerrors.New("oops"))
			return nil
		}, ch)
		err := <-ch
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
		require.Equal(t, "oops", panicErr.Err.Error())
	})

	t.Run("with another panic argument type", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			panic(33.7)
			return nil
		}, ch)
		err := <-ch
		require.Error(t, err)
		var panicErr *sqsafe.PanicError
		require.Error(t, err)
		require.True(t, xerrors.As(err, &panicErr))
		require.Equal(t, "33.7", panicErr.Err.Error())
	})

	t.Run("with a nil panic argument value", func(t *testing.T) {
		ch := make(chan error)
		sqsafe.Go(func() error {
			// This case cannot be differentiated yet.
			panic(nil)
			return xerrors.New("oops")
		}, ch)
		err := <-ch
		require.NoError(t, err)
	})
}

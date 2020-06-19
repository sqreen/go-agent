// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types_test

import (
	"testing"

	"github.com/sqreen/go-agent/sdk/types"
	"github.com/stretchr/testify/require"
	errors "golang.org/x/xerrors"
)

type myErrorType struct{ e error }

func (e myErrorType) Error() string { return e.e.Error() }
func (e myErrorType) Unwrap() error { return e.e }

func TestTypes(t *testing.T) {
	t.Run("errors", func(t *testing.T) {
		t.Run("properly implement the unwrap method", func(t *testing.T) {
			// Create a custom error type wrapping another one.
			// This error will be wrapped into the SqreenError type so that we can
			// test the Unwrap() method through the helper functions of package
			// `errors`.
			myLegacyErr := errors.New("my error")
			myErrWrapper := myErrorType{myLegacyErr}

			t.Run("through struct value", func(t *testing.T) {
				// Wrap myErrWrapper into SqreenError
				err := types.SqreenError{Err: myErrWrapper}

				// Unwrap down to SqreenError
				var sqerr types.SqreenError
				require.True(t, errors.As(err, &sqerr))

				// Unwrap down to myErrorType
				var myerr myErrorType
				require.True(t, errors.As(err, &myerr))

				// Unwrap down to myLegacyErr
				require.True(t, errors.Is(err, myLegacyErr))
			})

			t.Run("through pointer value", func(t *testing.T) {
				// Wrap myErrWrapper into a SqreenError pointer
				err := &types.SqreenError{Err: myErrWrapper}

				// Unwrap down to SqreenError with the wrong type
				var sqerrKO types.SqreenError
				require.False(t, errors.As(err, &sqerrKO))

				// Unwrap down to SqreenError with the correct type
				var sqerr *types.SqreenError
				require.True(t, errors.As(err, &sqerr))

				// Unwrap down to myErrorType
				var myerr myErrorType
				require.True(t, errors.As(err, &myerr))

				// Unwrap down to myLegacyErr
				require.True(t, errors.Is(err, myLegacyErr))
			})

		})
	})
}

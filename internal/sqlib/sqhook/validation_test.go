// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhook

import (
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook/internal"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestHookTableLookup(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		myIndex := symbolIndexType{}
		found, err := hookTableLookup(nil, testlib.RandUTF8String(), myIndex)
		require.NoError(t, err)
		require.Nil(t, found)
	})

	t.Run("empty", func(t *testing.T) {
		myIndex := symbolIndexType{}
		myTable := internal.HookTableType{}
		found, err := hookTableLookup(myTable, testlib.RandUTF8String(), myIndex)
		require.NoError(t, err)
		require.Nil(t, found)
	})

	t.Run("having instrumentation errors", func(t *testing.T) {
		myIndex := symbolIndexType{}
		for _, tc := range []internal.HookTableType{
			{
				func(d *internal.HookDescriptorType) {
					// Nil values
					*d = internal.HookDescriptorType{Func: nil, PrologVar: nil}
				},
			},

			{
				func(d *internal.HookDescriptorType) {
					// Nil Func value - Non-nil prolog var
					var prologVar *func()
					*d = internal.HookDescriptorType{Func: nil, PrologVar: &prologVar}
				},
			},
		} {
			tc := tc
			t.Run("", func(t *testing.T) {
				found, err := hookTableLookup(tc, testlib.RandUTF8String(), myIndex)
				require.Error(t, err)
				require.Nil(t, found)
			})
		}
	})
}

func TestPrologVarValidation(t *testing.T) {
	for _, tc := range []struct {
		fn, prolog    interface{}
		shouldSucceed bool
	}{
		{
			fn:            (func())(nil),
			prolog:        (func() (func(), error))(nil),
			shouldSucceed: true,
		},

		{ // wrong arg count
			fn:     (func())(nil),
			prolog: (func(*int) (func(), error))(nil),
		},

		{ // wrong prolog arg type: should be *int
			fn:     (func(int))(nil),
			prolog: (func(int) (func(), error))(nil),
		},

		{ // wrong prolog arg type: should be *int
			fn:            (func(int))(nil),
			prolog:        (func(*int) (func(), error))(nil),
			shouldSucceed: true,
		},

		{ // wrong return count
			fn:     (func(int))(nil),
			prolog: (func(*int) error)(nil),
		},

		{ // wrong return type: wrong prolog type
			fn:     (func(int))(nil),
			prolog: (func(*int) (func(string), error))(nil),
		},

		{ // wrong return type: wrong error type
			fn:     (func(int))(nil),
			prolog: (func(*int) (func(), bool))(nil),
		},

		{ // wrong return count
			fn:     (func(int))(nil),
			prolog: (func(*int) (func(), error, bool))(nil),
		},

		{ // wrong prolog type: wrong arg count
			fn:     (func(int) (chan struct{}, error))(nil),
			prolog: (func(*int) (func(*chan struct{}, *error, *int), error))(nil),
		},

		{ // wrong prolog type: wrong arg count
			fn:     (func(int) (chan struct{}, error))(nil),
			prolog: (func(*int) (func(*chan struct{}), error))(nil),
		},

		{ // wrong prolog type: wrong arg count
			fn:     (func(int) (chan struct{}, error))(nil),
			prolog: (func(*int) (func(), error))(nil),
		},

		{ // wrong prolog type: wrong arg types
			fn:     (func(int) (chan struct{}, error))(nil),
			prolog: (func(*int) (func(interface{}, interface{}, interface{}), error))(nil),
		},

		{ // wrong prolog type: wrong arg types
			fn:     (func(int) (chan struct{}, error))(nil),
			prolog: (func(*int) (func(chan struct{}, *error), error))(nil),
		},

		{
			fn:            (func(int) (chan struct{}, error))(nil),
			prolog:        (func(*int) (func(*chan struct{}, *error), error))(nil),
			shouldSucceed: true,
		},

		{ // variadic func
			fn:            (func(...int))(nil),
			prolog:        (func(*[]int) (func(), error))(nil),
			shouldSucceed: true,
		},
	} {
		tc := tc
		t.Run("unexpected signatures", func(t *testing.T) {
			fnType := reflect.TypeOf(tc.fn)
			prologVarType := reflect.PtrTo(reflect.PtrTo(reflect.TypeOf(tc.prolog)))
			err := validatePrologVar(fnType, prologVarType)
			if tc.shouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

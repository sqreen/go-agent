// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhook

import (
	"errors"
	"reflect"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook/internal"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func myFunction(int, string, bool) (float32, error) { return 0, nil }

func TestInstrumentationError(t *testing.T) {
	// Test that we would catch instrumentation mistakes - which should never
	// happen
	var myFunctionPrologVar *func(*int, *string, *bool) (func(*float32, *error), error)

	for _, tc := range []struct {
		Name      string
		Fn        interface{}
		PrologVar interface{}
	}{
		{
			Name:      "nil function",
			Fn:        nil,
			PrologVar: &myFunctionPrologVar,
		},

		{
			Name:      "not a function",
			Fn:        33,
			PrologVar: &myFunctionPrologVar,
		},

		{
			Name:      "nil prolog var",
			Fn:        myFunction,
			PrologVar: nil,
		},

		{
			Name:      "prolog var is not a pointer",
			Fn:        myFunction,
			PrologVar: myFunctionPrologVar,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			h, err := symbolIndexType{}.add(tc.Fn, tc.PrologVar)
			require.Error(t, err)
			require.Nil(t, h)
		})
	}
}

func TestReflectedCallback(t *testing.T) {
	t.Run("", func(t *testing.T) {
		type epilogType = func()
		type prologType = func() (epilogType, error)

		prolog, ok := makePrologCallback(&Hook{
			prologFuncType: reflect.TypeOf(prologType(nil)),
		}, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
			require.Len(t, params, 0)
			return nil, nil
		}).(prologType)

		require.True(t, ok)
		require.NotNil(t, prolog)

		epilog, err := prolog()
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

	t.Run("", func(t *testing.T) {
		type epilogType = func()
		type prologType = func(int, bool, string, float64, map[string]bool) (epilogType, error)

		var prologArgs struct {
			A int
			B bool
			C string
			D float64
			E map[string]bool
		}

		prolog, ok := makePrologCallback(&Hook{
			prologFuncType: reflect.TypeOf(prologType(nil)),
		}, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
			argValues := reflect.ValueOf(prologArgs)
			argTypes := argValues.Type()
			require.Len(t, params, argTypes.NumField())
			for i := range params {
				require.Equal(t, argTypes.Field(i).Type, params[i].Type())
				require.Equal(t, argValues.Field(i).Interface(), params[i].Interface())
			}
			return nil, nil
		}).(prologType)

		require.True(t, ok)
		require.NotNil(t, prolog)

		fuzz.New().Fuzz(&prologArgs)
		epilog, err := prolog(prologArgs.A, prologArgs.B, prologArgs.C, prologArgs.D, prologArgs.E)
		require.NoError(t, err)
		require.Nil(t, epilog)
	})

	t.Run("", func(t *testing.T) {
		type epilogType = func()
		type prologType = func(int, bool, string, float64, map[string]bool) (epilogType, error)

		var prologArgs struct {
			A int
			B bool
			C string
			D float64
			E map[string]bool
		}

		prologErr := errors.New("my error")

		prolog, ok := makePrologCallback(&Hook{
			prologFuncType: reflect.TypeOf(prologType(nil)),
		}, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
			argValues := reflect.ValueOf(prologArgs)
			argTypes := argValues.Type()
			require.Len(t, params, argTypes.NumField())
			for i := range params {
				require.Equal(t, argTypes.Field(i).Type, params[i].Type())
				require.Equal(t, argValues.Field(i).Interface(), params[i].Interface())
			}
			return nil, prologErr
		}).(prologType)

		require.True(t, ok)
		require.NotNil(t, prolog)

		fuzz.New().Fuzz(&prologArgs)
		epilog, err := prolog(prologArgs.A, prologArgs.B, prologArgs.C, prologArgs.D, prologArgs.E)
		require.Error(t, err)
		require.Equal(t, prologErr, err)
		require.Nil(t, epilog)
	})

	t.Run("", func(t *testing.T) {
		type epilogType = func(int, bool, string, float64, map[string]bool)
		type prologType = func(int, bool, string, float64, map[string]bool) (epilogType, error)

		var prologArgs, epilogArgs struct {
			A int
			B bool
			C string
			D float64
			E map[string]bool
		}

		prologErr := errors.New("my error")

		prolog, ok := makePrologCallback(&Hook{
			prologFuncType: reflect.TypeOf(prologType(nil)),
		}, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
			argValues := reflect.ValueOf(prologArgs)
			argTypes := argValues.Type()
			require.Len(t, params, argTypes.NumField())
			for i := range params {
				require.Equal(t, argTypes.Field(i).Type, params[i].Type())
				require.Equal(t, argValues.Field(i).Interface(), params[i].Interface())
			}
			return func(results []reflect.Value) {
				argValues := reflect.ValueOf(epilogArgs)
				argTypes := argValues.Type()
				require.Len(t, results, argTypes.NumField())
				for i := range results {
					require.Equal(t, argTypes.Field(i).Type, results[i].Type())
					require.Equal(t, argValues.Field(i).Interface(), results[i].Interface())
				}
			}, prologErr
		}).(prologType)

		require.True(t, ok)
		require.NotNil(t, prolog)

		f := fuzz.New()

		f.Fuzz(&prologArgs)
		epilog, err := prolog(prologArgs.A, prologArgs.B, prologArgs.C, prologArgs.D, prologArgs.E)
		require.Error(t, err)
		require.Equal(t, prologErr, err)

		require.NotNil(t, epilog)
		f.Fuzz(&epilogArgs)
		epilog(epilogArgs.A, epilogArgs.B, epilogArgs.C, epilogArgs.D, epilogArgs.E)
	})

	t.Run("", func(t *testing.T) {
		type epilogType = func(int, bool, string, float64, map[string]bool)
		type prologType = func(int, bool, string, float64, map[string]bool) (epilogType, error)

		var prologArgs, epilogArgs struct {
			A int
			B bool
			C string
			D float64
			E map[string]bool
		}

		prolog, ok := makePrologCallback(&Hook{
			prologFuncType: reflect.TypeOf(prologType(nil)),
		}, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
			argValues := reflect.ValueOf(prologArgs)
			argTypes := argValues.Type()
			require.Len(t, params, argTypes.NumField())
			for i := range params {
				require.Equal(t, argTypes.Field(i).Type, params[i].Type())
				require.Equal(t, argValues.Field(i).Interface(), params[i].Interface())
			}
			return func(results []reflect.Value) {
				argValues := reflect.ValueOf(epilogArgs)
				argTypes := argValues.Type()
				require.Len(t, results, argTypes.NumField())
				for i := range results {
					require.Equal(t, argTypes.Field(i).Type, results[i].Type())
					require.Equal(t, argValues.Field(i).Interface(), results[i].Interface())
				}
			}, nil
		}).(prologType)

		require.True(t, ok)
		require.NotNil(t, prolog)

		f := fuzz.New()

		f.Fuzz(&prologArgs)
		epilog, err := prolog(prologArgs.A, prologArgs.B, prologArgs.C, prologArgs.D, prologArgs.E)
		require.NoError(t, err)

		require.NotNil(t, epilog)
		f.Fuzz(&epilogArgs)
		epilog(epilogArgs.A, epilogArgs.B, epilogArgs.C, epilogArgs.D, epilogArgs.E)
	})
}

func TestMultiCallback(t *testing.T) {
	type epilogType = func(byte, rune, []string)
	type prologType = func(int, bool, string, float64, map[string]bool) (epilogType, error)

	var (
		prologArgs struct {
			A int
			B bool
			C string
			D float64
			E map[string]bool
		}
		epilogArgs struct {
			A byte
			B rune
			C []string
		}
		hook  = &Hook{prologFuncType: reflect.TypeOf(prologType(nil))}
		f     = fuzz.New()
		order []int
	)

	makePrologFunc := func(t *testing.T, expectedOrder int, epilog epilogType, prologErr error) prologType {
		return func(a int, b bool, c string, d float64, e map[string]bool) (epilogType, error) {
			require.Equal(t, prologArgs.A, a)
			require.Equal(t, prologArgs.B, b)
			require.Equal(t, prologArgs.C, c)
			require.Equal(t, prologArgs.D, d)
			require.Equal(t, prologArgs.E, e)
			order = append(order, expectedOrder)
			return epilog, prologErr
		}
	}

	makeEpilogFunc := func(t *testing.T, expectedOrder int) epilogType {
		return func(a byte, b rune, c []string) {
			require.Equal(t, epilogArgs.A, a)
			require.Equal(t, epilogArgs.B, b)
			require.Equal(t, epilogArgs.C, c)
			order = append(order, expectedOrder)
		}
	}

	for _, tc := range []struct {
		Prologs       []PrologCallback
		ExpectedOrder []int
		ExpectedError error
	}{
		{
			Prologs: []PrologCallback{
				makePrologFunc(t, 0, makeEpilogFunc(t, 11), nil),
				makePrologFunc(t, 1, makeEpilogFunc(t, 12), nil),
				makePrologFunc(t, 2, makeEpilogFunc(t, 13), nil),
				makePrologFunc(t, 3, makeEpilogFunc(t, 14), nil),
				makePrologFunc(t, 4, makeEpilogFunc(t, 15), nil),
				makePrologFunc(t, 5, makeEpilogFunc(t, 16), nil),
				makePrologFunc(t, 6, makeEpilogFunc(t, 17), nil),
				makePrologFunc(t, 7, makeEpilogFunc(t, 18), nil),
				makePrologFunc(t, 8, makeEpilogFunc(t, 19), nil),
				makePrologFunc(t, 9, makeEpilogFunc(t, 20), nil),
				makePrologFunc(t, 10, makeEpilogFunc(t, 21), nil),
			},
			ExpectedOrder: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21},
		},

		{
			Prologs: []PrologCallback{
				makePrologFunc(t, 0, makeEpilogFunc(t, 11), nil),
				makePrologFunc(t, 1, makeEpilogFunc(t, 12), nil),
				makePrologFunc(t, 2, makeEpilogFunc(t, 13), nil),
				makePrologFunc(t, 3, nil, nil),
				makePrologFunc(t, 4, makeEpilogFunc(t, 15), nil),
				makePrologFunc(t, 5, makeEpilogFunc(t, 16), nil),
				makePrologFunc(t, 6, nil, nil),
				makePrologFunc(t, 7, makeEpilogFunc(t, 18), nil),
				makePrologFunc(t, 8, nil, nil),
				makePrologFunc(t, 9, makeEpilogFunc(t, 20), nil),
				makePrologFunc(t, 10, makeEpilogFunc(t, 21), nil),
			},
			ExpectedOrder: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16, 18, 20, 21},
		},

		{
			Prologs: []PrologCallback{
				makePrologFunc(t, 0, makeEpilogFunc(t, 11), nil),
				makePrologFunc(t, 1, makeEpilogFunc(t, 12), nil),
				makePrologFunc(t, 2, makeEpilogFunc(t, 13), nil),
				makePrologFunc(t, 3, nil, nil),
				makePrologFunc(t, 4, makeEpilogFunc(t, 15), nil),
				makePrologFunc(t, 5, makeEpilogFunc(t, 16), errors.New("my error")),
				makePrologFunc(t, 6, nil, nil),
				makePrologFunc(t, 7, makeEpilogFunc(t, 18), nil),
				makePrologFunc(t, 8, nil, nil),
				makePrologFunc(t, 9, makeEpilogFunc(t, 20), nil),
				makePrologFunc(t, 10, makeEpilogFunc(t, 21), nil),
			},
			ExpectedOrder: []int{0, 1, 2, 3, 4, 5, 11, 12, 13, 15, 16},
			ExpectedError: errors.New("my error"),
		},

		{
			Prologs: []PrologCallback{
				makePrologFunc(t, 0, makeEpilogFunc(t, 11), errors.New("my error 1")),
				makePrologFunc(t, 1, makeEpilogFunc(t, 12), nil),
				makePrologFunc(t, 2, makeEpilogFunc(t, 13), nil),
				makePrologFunc(t, 3, nil, nil),
				makePrologFunc(t, 4, makeEpilogFunc(t, 15), nil),
				makePrologFunc(t, 5, makeEpilogFunc(t, 16), errors.New("my error 2")),
				makePrologFunc(t, 6, nil, nil),
				makePrologFunc(t, 7, makeEpilogFunc(t, 18), nil),
				makePrologFunc(t, 8, nil, nil),
				makePrologFunc(t, 9, makeEpilogFunc(t, 20), nil),
				makePrologFunc(t, 10, makeEpilogFunc(t, 21), nil),
			},
			ExpectedOrder: []int{0, 11},
			ExpectedError: errors.New("my error 1"),
		},

		{
			Prologs: []PrologCallback{
				makePrologFunc(t, 0, makeEpilogFunc(t, 11), nil),
				makePrologFunc(t, 1, makeEpilogFunc(t, 12), nil),
				makePrologFunc(t, 2, makeEpilogFunc(t, 13), nil),
				makePrologFunc(t, 3, nil, nil),
				makePrologFunc(t, 4, makeEpilogFunc(t, 15), nil),
				makePrologFunc(t, 5, makeEpilogFunc(t, 16), nil),
				makePrologFunc(t, 6, nil, nil),
				makePrologFunc(t, 7, makeEpilogFunc(t, 18), nil),
				makePrologFunc(t, 8, nil, nil),
				makePrologFunc(t, 9, makeEpilogFunc(t, 20), nil),
				makePrologFunc(t, 10, makeEpilogFunc(t, 21), errors.New("my error")),
			},
			ExpectedOrder: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 16, 18, 20, 21},
			ExpectedError: errors.New("my error"),
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			prolog, ok := makeMultiPrologCallback(hook, tc.Prologs).(prologType)
			require.True(t, ok)
			require.NotNil(t, prolog)

			f.Fuzz(&prologArgs)
			epilog, err := prolog(prologArgs.A, prologArgs.B, prologArgs.C, prologArgs.D, prologArgs.E)

			if tc.ExpectedError != nil {
				require.Error(t, err)
				require.Equal(t, tc.ExpectedError, err)
			} else {
				require.NoError(t, err)
			}

			require.NotNil(t, epilog)

			epilog(epilogArgs.A, epilogArgs.B, epilogArgs.C)

			require.Equal(t, tc.ExpectedOrder, order)
			order = nil // TODO: avoid this test side-effect...
		})
	}
}

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

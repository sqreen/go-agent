// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhook_test

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"unsafe"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

type example struct{}

func (example) method()                 {}
func (example) ExportedMethod()         {}
func (*example) methodPointerReceiver() {}

func function(_ int, _ string, _ bool) (float32, error) { return 0, nil }
func ExportedFunction(_ int, _ string, _ bool) error    { return nil }

func TestGoAssumptions(t *testing.T) {
	t.Run("getting a function pointer using reflect", func(t *testing.T) {
		var fn interface{} = function
		require.Equal(t, reflect.ValueOf(fn).Pointer(), reflect.ValueOf(function).Pointer())
	})

	t.Run("atomic store a function pointer", func(t *testing.T) {
		var cb *sqhook.PrologCallback
		addr := (*unsafe.Pointer)((unsafe.Pointer)(&cb))

		// Atomically store the function pointer
		var v sqhook.PrologCallback = function
		atomic.StorePointer(addr, unsafe.Pointer(&v))
		// Atomic load in order to ensure the sequential order of the memory 
		// accesses to &fn. Non-atomic reads could be otherwise reordered.
		atomicLoad := atomic.LoadPointer(addr) // sequential read barrier under the hood

		// Check
		require.Equal(t, unsafe.Pointer(&v), atomicLoad)
		require.Equal(t, &v, cb)
		require.True(t, reflect.TypeOf(*cb) == reflect.TypeOf(function))

		// Atomically store nil
		atomic.StorePointer(addr, nil)
		atomicLoad = atomic.LoadPointer(addr) // sequential read barrier under the hood

		// Check
		require.Equal(t, unsafe.Pointer(nil), atomicLoad)
		require.Equal(t, (*sqhook.PrologCallback)(nil), cb)
	})

	t.Run("the first argument of a method is the method receiver", func(t *testing.T) {
		require.Equal(t, reflect.TypeOf(example{}).Name(), reflect.TypeOf(example.method).In(0).Name())
	})

	t.Run("types can be compared using operator == on reflect.Type", func(t *testing.T) {
		val1 := example{}
		val2 := example{}
		val3 := &example{}
		require.True(t, reflect.TypeOf(val1) == reflect.TypeOf(val2))
		require.True(t, reflect.TypeOf(val1) != reflect.TypeOf(val3) && reflect.TypeOf(val3) == reflect.PtrTo(reflect.TypeOf(val1)))
	})
}

func TestNew(t *testing.T) {
	for _, tc := range []struct {
		value         interface{}
		shouldSucceed bool
	}{
		{func() {}, true},
		{nil, false},
		{(func())(nil), false},
		{example.method, true},
		{example.ExportedMethod, true},
		{(*example).methodPointerReceiver, true},
		{function, true},
		{ExportedFunction, true},
		{33, false},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T", tc.value), func(t *testing.T) {
			if tc.shouldSucceed {
				hook := sqhook.New(tc.value)
				require.NotNil(t, hook)
			} else {
				require.Panics(t, func() { sqhook.New(tc.value) })
			}
		})
	}
}

func TestFind(t *testing.T) {
	pkgName := reflect.TypeOf(example{}).PkgPath()
	for _, tc := range []struct {
		value  interface{}
		symbol string
	}{
		{example.method, "example.method"},
		{example.ExportedMethod, "example.ExportedMethod"},
		{(*example).methodPointerReceiver, "(*example).methodPointerReceiver"},
		{function, "function"},
		{ExportedFunction, "ExportedFunction"},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T", tc.value), func(t *testing.T) {
			hook := sqhook.New(tc.value)
			require.NotNil(t, hook)
			got := sqhook.Find(pkgName + "." + tc.symbol)
			require.NotNil(t, got)
			require.Equal(t, hook, got)
		})
	}
}

func TestAttach(t *testing.T) {
	for _, tc := range []struct {
		function, expected interface{}
		unexpected         []interface{}
	}{
		{
			function: func() {},
			expected: func() (func(), error) { return nil, nil },
			unexpected: []interface{}{
				"not even a function",
				func() (func(), int) { return nil, 33 },
				func() error { return nil },
				func() func() { return nil },
				func() {},
				func() (func() error, error) { return nil, nil },
			},
		},
		{
			function: example.method,
			expected: func(sqhook.MethodReceiver) (func(), error) { return nil, nil },
			unexpected: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) func() { return nil },
			},
		},
		{
			function: example.method,
			expected: func(*example) (func(), error) { return nil, nil },
			unexpected: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) func() { return nil },
			},
		},
		{
			function: example.ExportedMethod,
			expected: func(*example) (func(), error) { return nil, nil },
			unexpected: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) error { return nil },
			},
		},
		{
			function: (*example).methodPointerReceiver,
			expected: func(**example) (func(), error) { return nil, nil },
			unexpected: []interface{}{
				func() error { return nil },
				func() {},
				func(*example) (func(), error) { return nil, nil },
				func(example) (func(), error) { return nil, nil },
				func(**example) func() error { return nil },
				func(**example) error { return nil },
			},
		},
		{
			function: function,
			expected: func(*int, *string, *bool) (func(*float32, *error), error) { return nil, nil },
			unexpected: []interface{}{
				func(*int, *bool, *bool) error { return nil },
				func(*int, *string, *bool) error { return nil },
				func(*int, *string, *bool) {},
				func(int, string, bool) error { return nil },
				func(*int, *bool) error { return nil },
				func(*int) {},
				func(*int, *string, *bool) (func(*error), error) { return nil, nil },
				func(*int, *string, *bool) (func(float32, *error), error) { return nil, nil },
				func(*int, *string, bool) (func(*float32, *error), error) { return nil, nil },
			},
		},
		{
			function: ExportedFunction,
			expected: func(*int, *string, *bool) (func(*error), error) { return nil, nil },
			unexpected: []interface{}{
				func(*int, *bool, *bool) error { return nil },
				func(*int, *string, *bool) error { return nil },
				func(*int, *string, *bool) {},
				func(int, string, bool) error { return nil },
				func(*int, *bool) error { return nil },
				func(*int) {},
				func(*int, *string, *bool) (func(error), error) { return nil, nil },
				func(*int, string, *bool) (func(*error), error) { return nil, nil },
			},
		},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T", tc.function), func(t *testing.T) {
			t.Run("expected callback type", func(t *testing.T) {
				hook := sqhook.New(tc.function)
				require.NotNil(t, hook)
				err := hook.Attach(tc.expected)
				require.NoError(t, err)
				prolog := hook.Prolog()
				require.Equal(t, reflect.ValueOf(prolog).Pointer(), reflect.ValueOf(tc.expected).Pointer())
			})
			t.Run("not expected callback types", func(t *testing.T) {
				for _, notExpectedProlog := range tc.unexpected {
					notExpectedProlog := notExpectedProlog
					t.Run(fmt.Sprintf("%T", notExpectedProlog), func(t *testing.T) {
						hook := sqhook.New(tc.function)
						require.NotNil(t, hook)
						err := hook.Attach(notExpectedProlog)
						require.Error(t, err)
						prolog := hook.Prolog()
						require.Nil(t, prolog)
					})
				}
			})
		})
	}
}

func TestEnableDisable(t *testing.T) {
	hook := sqhook.New(example.ExportedMethod)
	require.NotNil(t, hook)
	err := hook.Attach(func(*example) (func(), error) { return nil, nil })
	require.NoError(t, err)
	prolog := hook.Prolog()
	require.NotNil(t, prolog)
	err = hook.Attach(nil)
	require.NoError(t, err)
	prolog = hook.Prolog()
	require.Nil(t, prolog)
}

func TestString(t *testing.T) {
	hook := sqhook.New(example.ExportedMethod)
	require.NotEmpty(t, hook.String())
}

func TestError(t *testing.T) {
	err := sqhook.AbortError
	require.NotEmpty(t, err.Error())
}

func TestUsage(t *testing.T) {
	t.Run("hooking a function and reading and writing the arguments and return values", func(t *testing.T) {
		var hook *sqhook.Hook

		// Fuzz the initial call arguments, and the arguments and return values the
		// callback will use to modify them.
		var (
			callA, expectedA int
			callB, expectedB string
			callC, expectedC bool
			callD, expectedD []byte
			expectedE        float32
			expectedF        error
		)
		fuzz := fuzz.New()
		fuzz.Fuzz(&callA)
		fuzz.Fuzz(&callB)
		fuzz.Fuzz(&callC)
		fuzz.Fuzz(&callD)
		fuzz.Fuzz(&expectedA)
		fuzz.Fuzz(&expectedB)
		fuzz.Fuzz(&expectedC)
		fuzz.Fuzz(&expectedD)
		fuzz.Fuzz(&expectedE)
		expectedF = errors.New("the error")

		example := func(a int, b string, c bool, d []byte) (e float32, f error) {
			{
				type Epilog = func(*float32, *error)
				type Prolog = func(*int, *string, *bool, *[]byte) (Epilog, error)
				prolog := hook.Prolog()

				if prolog, ok := prolog.(Prolog); ok {
					epilog, err := prolog(&a, &b, &c, &d)
					if epilog != nil {
						defer epilog(&e, &f)
					}
					if err != nil {
						return
					}
				}
			}
			// Check the arguments were modified
			require.Equal(t, expectedA, a)
			require.Equal(t, expectedB, b)
			require.Equal(t, expectedC, c)
			require.Equal(t, expectedD, d)
			// Return some values that should get modified by the epilog callback
			return 33, nil
		}

		// Define a hook a attach prolog and epilog callbacks that will modify the
		// arguments and return values
		hook = sqhook.New(example)
		require.NotNil(t, hook)
		err := hook.Attach(
			func(a *int, b *string, c *bool, d *[]byte) (func(*float32, *error), error) {
				require.Equal(t, callA, *a)
				require.Equal(t, callB, *b)
				require.Equal(t, callC, *c)
				require.Equal(t, callD, *d)
				// Modify the arguments
				*a = expectedA
				*b = expectedB
				*c = expectedC
				*d = expectedD
				return func(e *float32, f *error) {
					// Modify the return values
					*e = expectedE
					*f = expectedF
				}, nil
			})
		require.NoError(t, err)

		e, f := example(callA, callB, callC, callD)
		// Check the returned values were also modified
		require.Equal(t, expectedE, e)
		require.Equal(t, expectedF, f)
	})
}

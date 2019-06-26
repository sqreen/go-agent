// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhook_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

type example struct{}

func (example) method()                 {}
func (example) ExportedMethod()         {}
func (*example) methodPointerReceiver() {}

func function(_ int, _ string, _ bool) error         { return nil }
func ExportedFunction(_ int, _ string, _ bool) error { return nil }

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
			hook := sqhook.New(tc.value)
			if tc.shouldSucceed {
				require.NotNil(t, hook)
			} else {
				require.Nil(t, hook)
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
		function, expectedProlog, expectedEpilog interface{}
		notExpectedPrologs, notExpectedEpilogs   []interface{}
	}{
		{
			function:       func() {},
			expectedProlog: func(*sqhook.Context) error { return nil },
			expectedEpilog: func(*sqhook.Context) {},
			notExpectedPrologs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) {},
				func() {},
			},
			notExpectedEpilogs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) error { return nil },
				func() {},
			},
		},
		{
			function:       example.method,
			expectedProlog: func(*sqhook.Context) error { return nil },
			expectedEpilog: func(*sqhook.Context) {},
			notExpectedPrologs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) {},
				func() {},
			},
			notExpectedEpilogs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) error { return nil },
				func() {},
			},
		},
		{
			function:       example.ExportedMethod,
			expectedProlog: func(*sqhook.Context) error { return nil },
			expectedEpilog: func(*sqhook.Context) {},
			notExpectedPrologs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) {},
				func() {},
			},
			notExpectedEpilogs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) error { return nil },
				func() {},
			},
		},
		{
			function:       (*example).methodPointerReceiver,
			expectedProlog: func(*sqhook.Context) error { return nil },
			expectedEpilog: func(*sqhook.Context) {},
			notExpectedPrologs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) {},
				func() {},
			},
			notExpectedEpilogs: []interface{}{
				func() error { return nil },
				func(*sqhook.Context) error { return nil },
				func() {},
			},
		},
		{
			function:       function,
			expectedProlog: func(*sqhook.Context, *int, *string, *bool) error { return nil },
			expectedEpilog: func(*sqhook.Context, *error) {},
			notExpectedPrologs: []interface{}{
				func(*sqhook.Context, *int, *bool, *bool) error { return nil },
				func(*int, *string, *bool) error { return nil },
				func(*sqhook.Context, *int, *string, *bool) {},
				func(*sqhook.Context, int, string, bool) error { return nil },
				func(*sqhook.Context, *int, *bool) error { return nil },
				func(*sqhook.Context, *int) {},
			},
			notExpectedEpilogs: []interface{}{
				func(*error) {},
				func(*sqhook.Context, error) {},
				func() {},
			},
		},
		{
			function:       ExportedFunction,
			expectedProlog: func(*sqhook.Context, *int, *string, *bool) error { return nil },
			expectedEpilog: func(*sqhook.Context, *error) {},
			notExpectedPrologs: []interface{}{
				func(*sqhook.Context, *int, *string, *string) error { return nil },
				func(*int, *string, *bool) error { return nil },
				func(*sqhook.Context, *int, *string, *bool) {},
				func(*sqhook.Context, int, string, bool) error { return nil },
				func(*sqhook.Context, *int, *bool) error { return nil },
			},
			notExpectedEpilogs: []interface{}{
				func(*error) {},
				func(*sqhook.Context, error) {},
				func(*sqhook.Context, *int) {},
				func() {},
			},
		},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%T", tc.function), func(t *testing.T) {
			t.Run("expected callbacks", func(t *testing.T) {
				t.Run("non-nil prolog and epilog", func(t *testing.T) {
					hook := sqhook.New(tc.function)
					require.NotNil(t, hook)
					err := hook.Attach(tc.expectedProlog, tc.expectedEpilog)
					require.NoError(t, err)
					prolog, epilog := hook.Callbacks()
					require.Equal(t, reflect.ValueOf(prolog).Pointer(), reflect.ValueOf(tc.expectedProlog).Pointer())
					require.Equal(t, reflect.ValueOf(epilog).Pointer(), reflect.ValueOf(tc.expectedEpilog).Pointer())
				})
				t.Run("nil prolog", func(t *testing.T) {
					hook := sqhook.New(tc.function)
					require.NotNil(t, hook)
					err := hook.Attach(nil, tc.expectedEpilog)
					require.NoError(t, err)
					prolog, epilog := hook.Callbacks()
					require.Nil(t, prolog)
					require.Equal(t, reflect.ValueOf(epilog).Pointer(), reflect.ValueOf(tc.expectedEpilog).Pointer())
				})
				t.Run("nil epilog", func(t *testing.T) {
					hook := sqhook.New(tc.function)
					require.NotNil(t, hook)
					err := hook.Attach(tc.expectedProlog, nil)
					require.NoError(t, err)
					prolog, epilog := hook.Callbacks()
					require.Equal(t, reflect.ValueOf(prolog).Pointer(), reflect.ValueOf(tc.expectedProlog).Pointer())
					require.Nil(t, epilog)
				})
				t.Run("nil prolog and epilog", func(t *testing.T) {
					hook := sqhook.New(tc.function)
					require.NotNil(t, hook)
					err := hook.Attach(nil, nil)
					require.NoError(t, err)
					prolog, epilog := hook.Callbacks()
					require.Nil(t, prolog)
					require.Nil(t, epilog)
				})
			})
			t.Run("not expected callbacks", func(t *testing.T) {
				for _, notExpectedProlog := range tc.notExpectedPrologs {
					notExpectedProlog := notExpectedProlog
					t.Run(fmt.Sprintf("%T", notExpectedProlog), func(t *testing.T) {
						hook := sqhook.New(tc.function)
						require.NotNil(t, hook)
						err := hook.Attach(notExpectedProlog, tc.expectedEpilog)
						require.Error(t, err)
						prolog, epilog := hook.Callbacks()
						require.Nil(t, prolog)
						require.Nil(t, epilog)
					})
				}
				for _, notExpectedEpilog := range tc.notExpectedEpilogs {
					notExpectedEpilog := notExpectedEpilog
					t.Run(fmt.Sprintf("%T", notExpectedEpilog), func(t *testing.T) {
						hook := sqhook.New(tc.function)
						require.NotNil(t, hook)
						err := hook.Attach(tc.expectedProlog, notExpectedEpilog)
						require.Error(t, err)
						prolog, epilog := hook.Callbacks()
						require.Nil(t, prolog)
						require.Nil(t, epilog)
					})
				}
			})
		})
	}
}

func TestEnableDisable(t *testing.T) {
	hook := sqhook.New(example.ExportedMethod)
	require.NotNil(t, hook)
	err := hook.Attach(func(*sqhook.Context) error { return nil }, func(*sqhook.Context) {})
	require.NoError(t, err)
	prolog, epilog := hook.Callbacks()
	require.NotNil(t, prolog)
	require.NotNil(t, epilog)
	hook.Attach(nil, nil)
	prolog, epilog = hook.Callbacks()
	require.Nil(t, prolog)
	require.Nil(t, epilog)
}

func TestUsage(t *testing.T) {
	t.Run("nil hook", func(t *testing.T) {
		hook := sqhook.New(33)
		require.Nil(t, hook)
		err := hook.Attach("oops", "no")
		require.Error(t, err)
		prolog, epilog := hook.Callbacks()
		require.Nil(t, prolog)
		require.Nil(t, epilog)
	})

	t.Run("attaching nil", func(t *testing.T) {
		hook := sqhook.New(example.method)
		require.NotNil(t, hook)
		err := hook.Attach(func(*sqhook.Context) error { return nil }, nil)
		require.NoError(t, err)
		prolog, epilog := hook.Callbacks()
		require.NotNil(t, prolog)
		require.Nil(t, epilog)
		err = hook.Attach(nil, nil)
		require.NoError(t, err)
		prolog, epilog = hook.Callbacks()
		require.Nil(t, prolog)
		require.Nil(t, epilog)
	})

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
				type Prolog = func(*sqhook.Context, *int, *string, *bool, *[]byte) error
				type Epilog = func(*sqhook.Context, *float32, *error)
				ctx := sqhook.Context{}
				prolog, epilog := hook.Callbacks()
				if epilog, ok := epilog.(Epilog); ok {
					defer epilog(&ctx, &e, &f)
				}
				if prolog, ok := prolog.(Prolog); ok {
					err := prolog(&ctx, &a, &b, &c, &d)
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
			func(ctx *sqhook.Context, a *int, b *string, c *bool, d *[]byte) error {
				require.Equal(t, callA, *a)
				require.Equal(t, callB, *b)
				require.Equal(t, callC, *c)
				require.Equal(t, callD, *d)
				// Modify the arguments
				*a = expectedA
				*b = expectedB
				*c = expectedC
				*d = expectedD
				return nil
			},
			func(ctx *sqhook.Context, e *float32, f *error) {
				// Modify the return values
				*e = expectedE
				*f = expectedF
			})
		require.NoError(t, err)

		e, f := example(callA, callB, callC, callD)
		// Check the returned values were also modified
		require.Equal(t, expectedE, e)
		require.Equal(t, expectedF, f)
	})
}

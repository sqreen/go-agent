// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhook_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/require"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook/internal"
)

// Mock the instrumentation tool

//go:linkname _sqreen_instrumentation_descriptor _sqreen_instrumentation_descriptor
var _sqreen_instrumentation_descriptor *internal.InstrumentationDescriptorType

var MyMethodProlog *func(*example) (func(), error)
var MyExportedMethodProlog *func(*example) (func(), error)
var MyMethodWithPointerRecvProlog *func(**example) (func(), error)
var MyFunctionProlog *func(*int, *string, *bool) (func(*float32, *error), error)
var MyExportedFunctionProlog *func(*int, *string, *bool) (func(*error), error)

type example struct{}

func (example) myMethod()                     {}
func (example) MyExportedMethod()             {}
func (*example) myMethodWithPointerReceiver() {}

func myFunction(_ int, _ string, _ bool) (float32, error)  { return 0, nil }
func myFunction2(_ int, _ string, _ bool) (float32, error) { return 0, nil }
func MyExportedFunction(_ int, _ string, _ bool) error     { return nil }

var (
	MyMethodSymbol                = runtime.FuncForPC(reflect.ValueOf(example.myMethod).Pointer()).Name()
	MyMethodWithPointerRecvSymbol = runtime.FuncForPC(reflect.ValueOf((*example).myMethodWithPointerReceiver).Pointer()).Name()
	MyExportedMethodSymbol        = runtime.FuncForPC(reflect.ValueOf(example.MyExportedMethod).Pointer()).Name()
	MyFunctionSymbol              = runtime.FuncForPC(reflect.ValueOf(myFunction).Pointer()).Name()
	MyFunction2Symbol             = runtime.FuncForPC(reflect.ValueOf(myFunction2).Pointer()).Name()
	MyExportedFunctionSymbol      = runtime.FuncForPC(reflect.ValueOf(MyExportedFunction).Pointer()).Name()
)

var sortedSymbols = []string{ // Sorted by normalized name
	MyExportedFunctionSymbol,
	MyExportedMethodSymbol,
	MyMethodSymbol,
	MyMethodWithPointerRecvSymbol,
	MyFunctionSymbol,
	MyFunction2Symbol,
}

var expectedSymbols = map[string]internal.HookDescriptorFuncType{
	MyMethodSymbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      example.myMethod,
			PrologVar: &MyMethodProlog,
		}
	},

	MyMethodWithPointerRecvSymbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      (*example).myMethodWithPointerReceiver,
			PrologVar: &MyMethodWithPointerRecvProlog,
		}
	},

	MyExportedMethodSymbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      example.MyExportedMethod,
			PrologVar: &MyExportedMethodProlog,
		}
	},

	MyFunctionSymbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      myFunction,
			PrologVar: &MyFunctionProlog,
		}
	},

	MyFunction2Symbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      myFunction2,
			PrologVar: &MyFunctionProlog,
		}
	},

	MyExportedFunctionSymbol: func(d *internal.HookDescriptorType) {
		*d = internal.HookDescriptorType{
			Func:      MyExportedFunction,
			PrologVar: &MyExportedFunctionProlog,
		}
	},
}

func init() {
	_sqreen_instrumentation_descriptor = &internal.InstrumentationDescriptorType{
		Version:   "1.2.3",
		HookTable: make(internal.HookTableType, len(sortedSymbols)),
	}
	for i, sym := range sortedSymbols {
		_sqreen_instrumentation_descriptor.HookTable[i] = expectedSymbols[sym]
	}
}

// !Mock the instrumentation tool

func TestGoAssumptions(t *testing.T) {
	t.Run("getting a function pointer using reflect", func(t *testing.T) {
		var fn interface{} = myFunction
		require.Equal(t, reflect.ValueOf(fn).Pointer(), reflect.ValueOf(myFunction).Pointer())
	})

	t.Run("atomic store a function pointer", func(t *testing.T) {
		var cb *sqhook.PrologCallback
		addr := (*unsafe.Pointer)((unsafe.Pointer)(&cb))

		// Atomically store the function pointer
		var v sqhook.PrologCallback = myFunction
		atomic.StorePointer(addr, unsafe.Pointer(&v))
		// Atomic load in order to ensure the sequential order of the memory
		// accesses to &fn. Non-atomic reads could be otherwise reordered.
		atomicLoad := atomic.LoadPointer(addr) // sequential read barrier under the hood

		// Check
		require.Equal(t, unsafe.Pointer(&v), atomicLoad)
		require.Equal(t, &v, cb)
		require.True(t, reflect.TypeOf(*cb) == reflect.TypeOf(myFunction))

		// Atomically store nil
		atomic.StorePointer(addr, nil)
		atomicLoad = atomic.LoadPointer(addr) // sequential read barrier under the hood

		// Check
		require.Equal(t, unsafe.Pointer(nil), atomicLoad)
		require.Equal(t, (*sqhook.PrologCallback)(nil), cb)
	})

	t.Run("the first argument of a method is the method receiver", func(t *testing.T) {
		require.Equal(t, reflect.TypeOf(example{}).Name(), reflect.TypeOf(example.myMethod).In(0).Name())
	})

	t.Run("types can be compared using operator == on reflect.Type", func(t *testing.T) {
		val1 := example{}
		val2 := example{}
		val3 := &example{}
		require.True(t, reflect.TypeOf(val1) == reflect.TypeOf(val2))
		require.True(t, reflect.TypeOf(val1) != reflect.TypeOf(val3) && reflect.TypeOf(val3) == reflect.PtrTo(reflect.TypeOf(val1)))
	})

	t.Run("Pointer method of a pointer gives the pointer address", func(t *testing.T) {
		var (
			v int
			i interface{} = &v
		)
		require.Equal(t, reflect.ValueOf(i).Pointer(), uintptr(unsafe.Pointer(&v)))
	})

}

func TestFind(t *testing.T) {
	for sym := range expectedSymbols {
		sym := sym
		t.Run(sym, func(t *testing.T) {
			hook, err := sqhook.Find(sym)
			require.NoError(t, err)
			require.NotNil(t, hook)
		})
	}
}

type prologCallbackGetter struct {
	prolog sqhook.PrologCallback
}

func (p prologCallbackGetter) PrologCallback() sqhook.PrologCallback {
	return p.prolog
}

func TestAttach(t *testing.T) {
	for _, tc := range []struct {
		Symbol         string
		InvalidPrologs []interface{}
	}{
		{
			Symbol: MyFunctionSymbol,
			InvalidPrologs: []interface{}{
				"not even a function",
				func() (func(), int) { return nil, 33 },
				func() error { return nil },
				func() func() { return nil },
				func() {},
				func() (func() error, error) { return nil, nil },
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
			Symbol: MyMethodSymbol,
			InvalidPrologs: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) func() { return nil },
			},
		},
		{
			Symbol: MyMethodWithPointerRecvSymbol,
			InvalidPrologs: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) func() { return nil },
			},
		},
		{
			Symbol: MyExportedMethodSymbol,
			InvalidPrologs: []interface{}{
				func() error { return nil },
				func() {},
				func(example) (func(), error) { return nil, nil },
				func(*example) (func() error, error) { return nil, nil },
				func(*example) error { return nil },
				func(example) (func(), error) { return nil, nil },
				func(**example) func() error { return nil },
				func(**example) error { return nil },
			},
		},
		{
			Symbol: MyExportedFunctionSymbol,
			InvalidPrologs: []interface{}{
				func() error { return nil },
				func() {},
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

		// Create the expected prolog function out of the hook descriptor of the
		// symbol
		var descr internal.HookDescriptorType
		getDescr := expectedSymbols[tc.Symbol]
		getDescr(&descr)
		expectedPrologType := reflect.TypeOf(descr.PrologVar).Elem().Elem()
		expectedProlog := reflect.MakeFunc(expectedPrologType, func(args []reflect.Value) (results []reflect.Value) {
			return []reflect.Value{{}, {}} // not used by the test
		})

		checkPrologAddr := func(t *testing.T, expected uintptr) {
			// Read barrier using the prolog var
			_ = atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(reflect.ValueOf(descr.PrologVar).Pointer())))
			require.Equal(t, expected, reflect.ValueOf(descr.PrologVar).Elem().Elem().Pointer())
		}

		checkPrologAddrNotNil := func(t *testing.T) {
			// Read barrier using the prolog var
			_ = atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(reflect.ValueOf(descr.PrologVar).Pointer())))
			require.NotZero(t, reflect.ValueOf(descr.PrologVar).Elem().Elem().Pointer())
		}

		t.Run(tc.Symbol, func(t *testing.T) {
			t.Parallel() // can run in parallel

			t.Run("expected prolog type", func(t *testing.T) {
				// Find the hook
				hook, err := sqhook.Find(tc.Symbol)
				require.NoError(t, err)
				require.NotNil(t, hook)

				t.Run("native prolog callback", func(t *testing.T) {
					// Attach the expected prolog function
					err = hook.Attach(expectedProlog.Interface())
					require.NoError(t, err)
					// Read back the prolog variable
					checkPrologAddr(t, expectedProlog.Pointer())
				})

				t.Run("reflected prolog callback", func(t *testing.T) {
					var reflected sqhook.ReflectedPrologCallback = func(params []reflect.Value) (epilog sqhook.ReflectedEpilogCallback, err error) {
						return nil, nil
					}
					// Attach the expected prolog function
					err = hook.Attach(reflected)
					require.NoError(t, err)
					// Read back the prolog variable
					checkPrologAddrNotNil(t)
				})

				t.Run("prolog callback getter", func(t *testing.T) {
					t.Run("returning a native prolog", func(t *testing.T) {
						// Attach the expected prolog function
						err = hook.Attach(prologCallbackGetter{prolog: expectedProlog.Interface()})
						require.NoError(t, err)
						// Read back the prolog variable
						checkPrologAddrNotNil(t)
					})

					t.Run("returning a reflected prolog", func(t *testing.T) {
						var reflected sqhook.ReflectedPrologCallback = func(params []reflect.Value) (epilog sqhook.ReflectedEpilogCallback, err error) {
							return nil, nil
						}
						// Attach the expected prolog function
						err = hook.Attach(prologCallbackGetter{
							prolog: reflected,
						})
						require.NoError(t, err)
						// Read back the prolog variable
						checkPrologAddrNotNil(t)
					})
				})

				t.Run("multiple prolog callbacks", func(t *testing.T) {
					var reflected sqhook.ReflectedPrologCallback = func(params []reflect.Value) (epilog sqhook.ReflectedEpilogCallback, err error) {
						return nil, nil
					}
					// Attach the expected prolog function
					native := expectedProlog.Interface()
					err = hook.Attach(reflected, native, reflected, native, reflected, native, prologCallbackGetter{prolog: reflected}, prologCallbackGetter{prolog: expectedProlog.Interface()})
					require.NoError(t, err)
					// Read back the prolog variable
					checkPrologAddrNotNil(t)
				})
			})

			t.Run("not expected prolog types", func(t *testing.T) {
				hook, err := sqhook.Find(tc.Symbol)
				require.NoError(t, err)
				require.NotNil(t, hook)
				require.NoError(t, hook.Attach(nil))

				for _, invalidProlog := range tc.InvalidPrologs {
					invalidProlog := invalidProlog
					t.Run(fmt.Sprintf("%T", invalidProlog), func(t *testing.T) {
						err = hook.Attach(invalidProlog)
						require.Error(t, err)
						//checkPrologAddr(t, 0)
					})

					t.Run(fmt.Sprintf("%T along with the expected prolog callback", invalidProlog), func(t *testing.T) {
						err = hook.Attach(expectedProlog.Interface(), invalidProlog)
						require.Error(t, err)
						//checkPrologAddr(t, 0)

						err = hook.Attach(invalidProlog, expectedProlog.Interface())
						require.Error(t, err)
						//checkPrologAddr(t, 0)
					})
				}
			})
		})
	}
}

func TestEnableDisable(t *testing.T) {
	// Pick a random value symbol
	sym := sortedSymbols[rand.Intn(len(sortedSymbols))]

	// Find it
	hook, err := sqhook.Find(sym)
	require.NoError(t, err)
	require.NotNil(t, hook)

	// Attach the expected prolog type
	var descr internal.HookDescriptorType
	getDescr := expectedSymbols[sym]
	getDescr(&descr)
	prologType := reflect.TypeOf(descr.PrologVar).Elem().Elem()
	expectedProlog := reflect.MakeFunc(prologType, func(args []reflect.Value) (results []reflect.Value) {
		return []reflect.Value{{}, {}}
	})
	err = hook.Attach(expectedProlog.Interface())
	require.NoError(t, err)

	// Read the prolog variable and check it points to the previous prolog
	// function
	loadProlog := func() unsafe.Pointer {
		// Atomic load of the prolog var
		return atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(reflect.ValueOf(descr.PrologVar).Pointer())))
	}
	prolog := loadProlog()
	require.NotNil(t, prolog)
	// Use the typed prolog var to check the pointer value. The loadProlog call
	// acts like a barrier to make sure the code isn't reordered.
	require.Equal(t, expectedProlog.Pointer(), reflect.ValueOf(descr.PrologVar).Elem().Elem().Pointer())

	// Disable it by attaching nil
	err = hook.Attach(nil)
	require.NoError(t, err)
	// Walk the prolog var value in order to get the function pointer
	prolog = loadProlog()
	require.Equal(t, unsafe.Pointer(nil), prolog)
}

func TestStringer(t *testing.T) {
	for sym := range expectedSymbols {
		sym := sym
		t.Run(sym, func(t *testing.T) {
			hook, err := sqhook.Find(sym)
			require.NoError(t, err)
			require.NotNil(t, hook)
			require.NotEmpty(t, hook.String())
		})
	}
}

func TestError(t *testing.T) {
	err := sqhook.AbortError
	require.NotEmpty(t, err.Error())
}

func TestHealth(t *testing.T) {
	require.NoError(t, sqhook.Health("1.2.3"))
	require.Error(t, sqhook.Health(""))
	require.Error(t, sqhook.Health("1.2.4"))
}

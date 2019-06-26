// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Package sqhook provides a pure Go implementation of hooks to be inserted
// into function definitions in order to be able to attach prolog and epilog
// callbacks giving read/write access to the arguments and returned values of
// the function call at run time.
//
// A hook needs to be globally created and associated to a function symbol at
// package initialization time. Prolog and epilog callbacks can then be
// accessed in the function call to pass the call arguments and return values.
//
// On the other side, callbacks can be attached to a hook at run time. Prolog
// callbacks get read/write access to the arguments of the function call before
// it gets executed, while epilog callbacks get read/write access to the return
// values before returning from the call. Therefore, the callbacks' signature
// need to match the function signature.
//
// Given a function F:
// 		func F(A, B, C) (R, S, T)
// The expected prolog signature is:
//		type prolog = func(*sqhook.Context, *A, *B, *C) error
// The expected epilog signature is:
//		type epilog = func(*sqhook.Context, *R, *S, *T)
//
// Example:
//		// Define the hook globally
//		var exampleHook *sqhook.Hook
//
//		// Initialization needs to be done in the init() function because of some
//		// Go initialization limitations.
// 		func init() {
// 			exampleHook = sqhook.New(Example)
// 		}
//
// 		func Example(arg1 int, arg2 string) (ret1 []byte, ret2 error) {
//			// Use the hook first and call its callbacks
//			{
//				type Prolog = func(*sqhook.Context, *int, *string) error
//				type Epilog = func(*sqhook.Context, *[]byte, *error)
//				// Create a call context
//				ctx := sqhook.Context{}
//				prolog, epilog := exampleHook.Callbacks()
//				// If an epilog is set, defer the call to the epilog
//				if epilog, ok := epilog.(Epilog); ok {
//					// Pass pointers to the return values
//					defer epilog(&ctx, &ret1, &ret2)
//				}
//				// If a prolog is set, call it
//				if prolog, ok := prolog.(Prolog); ok {
//					// Pass pointers to the arguments
//					err := prolog(&ctx, &w, &r, &headers, &statusCode, &body)
//					// If an error is returned, the function execution is aborted.
//					// The deferred epilog call will still be executed before returning.
//					if err != nil {
//						return
//					}
//				}
//			}
// 			// .. function code ...
//		}
//
//
// Main requirements:
// - Concurrent access and modification of callbacks.
// - Reentrant implementation of callbacks with a call context when data needs
//   to be shared between the prolog and epilog.
//
// - Fast call dispatch for callbacks that don't need to be generic, ie.
//   callbacks that are designed to be attached to specific functions.
//   Type-assertion instead of `reflect.Call` is therefore used while generic
//   callbacks that are not tied a specific function will be attached using
//   `reflect.MakeFunc` in order to match the function signature. The usage
//   of dynamic calls using `reflect` is indeed much slower and consumes
//   memory.
//
// Design constraints:
//
// - There are no compilation-time functions or macros that would have allowed
//   to provide helpers setting up the hooks in the function definitions.
// - Access and modification of callbacks need to be atomic.
// - There are no way to add custom sections to the binary file, which would
//   have made possible defining the index of hooks at compilation-time (
//   things that can be easily done with GCC).
//
package sqhook

import (
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

var index = make(map[string]*Hook)

type Hook struct {
	// The function type where the hook is used.
	fnType reflect.Type
	// Pointer to a structure containing the callbacks in order to be able to
	// atomically modify the pointer.
	attached *callbacks
}

type callbacks struct {
	prolog, epilog Callback
}

// Callback is a function expecting a Context pointer as first argument,
// followed by the pointers to the arguments of the hooked function for a
// prolog, and followed by the pointers to the returned values for an epilog.
type Callback interface{}

// Context is a call context for the hook. It is shared between the prolog and
// epilog and is unique for each function call. It allows callbacks to provide
// reentrant implementations when memory needs to be shared for a given call.
type Context []interface{}

// MethodReceiver is store in the context when hooking a method.
type MethodReceiver interface{}

type Error int

// Errors that hooks can return in order to modify the control flow of the
// function.
const (
	_ Error = iota
	// Abort the execution of the function by returning from it.
	AbortError
)

func (e Error) Error() string {
	return fmt.Sprintf("Error(%d)", e)
}

// Static assertion that `Error` implements interface `error`
var _ error = Error(0)

// New returns a hook for function `fn` to be used in the function definition
// in order to be able to attach callbacks to it. It returns nil if the fn is
// not a non-nil function or if the symbol name of `fn` cannot be retrieved.
func New(fn interface{}) *Hook {
	// Check fn is a non-nil function value.
	if fn == nil {
		return nil
	}
	v := reflect.ValueOf(fn)
	fnType := v.Type()
	if fnType.Kind() != reflect.Func {
		return nil
	}
	// If the symbol name cannot be retrieved
	symbol := runtime.FuncForPC(v.Pointer()).Name()
	if symbol == "" {
		return nil
	}
	// Create the hook, store it in the map and return it.
	hook := &Hook{
		fnType: fnType,
	}
	index[symbol] = hook
	return hook
}

// Find returns the hook associated to the given symbol string when it was
// created using `New()`, nil otherwise.
func Find(symbol string) *Hook {
	return index[symbol]
}

// Attach atomically attaches prolog and epilog callbacks to the hook. It is
// possible to pass nil values when only one type of callback is required. If
// both arguments are nil, the callbacks are removed.
func (h *Hook) Attach(prolog, epilog Callback) error {
	if h == nil {
		return sqerrors.New("cannot attach callbacks to a nil hook")
	}
	var cbs *callbacks
	if prolog != nil || epilog != nil {
		cbs = &callbacks{}
		if prolog != nil {
			// Create the list of argument types
			argTypes := make([]reflect.Type, 0, h.fnType.NumIn())
			for i := 0; i < h.fnType.NumIn(); i++ {
				argTypes = append(argTypes, h.fnType.In(i))
			}
			if err := validateProlog(prolog, argTypes); err != nil {
				return err
			}
			cbs.prolog = prolog
		}
		if epilog != nil {
			// Create the list of return types
			retTypes := make([]reflect.Type, 0, h.fnType.NumOut())
			for i := 0; i < h.fnType.NumOut(); i++ {
				retTypes = append(retTypes, h.fnType.Out(i))
			}
			if err := validateEpilog(epilog, retTypes); err != nil {
				return err
			}
			cbs.epilog = epilog
		}
	}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&h.attached)), unsafe.Pointer(cbs))
	return nil
}

// Callbacks atomically accesses the attached prolog and epilog callbacks.
func (h *Hook) Callbacks() (prolog, epilog Callback) {
	if h == nil {
		return nil, nil
	}
	attached := (*callbacks)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&h.attached))))
	if attached == nil {
		return nil, nil
	}
	return attached.prolog, attached.epilog
}

// validateProlog validates that the prolog has the expected signature.
func validateProlog(prolog Callback, argTypes []reflect.Type) error {
	if err := validateCallback(prolog, argTypes); err != nil {
		return err
	}
	t := reflect.TypeOf(prolog)
	if t.NumOut() != 1 || !t.Out(0).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return sqerrors.New("validation: the prolog callback should return an error value")
	}
	return nil
}

// validateEpilog validates that the epilog has the expected signature.
func validateEpilog(epilog Callback, argTypes []reflect.Type) error {
	if err := validateCallback(epilog, argTypes); err != nil {
		return err
	}
	t := reflect.TypeOf(epilog)
	if t.NumOut() != 0 {
		return sqerrors.New("validation: the epilog callback should not return values")
	}
	return nil
}

// validateCallback validates the fact that the callback is a function whose
// first argument is the hook context and the rest of its arguments can be
// assigned the hook argument values.
func validateCallback(callback Callback, argTypes []reflect.Type) (err error) {
	defer func() {
		if err != nil {
			err = sqerrors.Wrap(err, "validation error")
		}
	}()
	callbackType := reflect.TypeOf(callback)
	// Check the callback is a function
	if callbackType.Kind() != reflect.Func {
		return sqerrors.New("the callback argument is not a function")
	}
	callbackArgc := callbackType.NumIn()
	// Check the callback accepts a hook context as first argument
	if callbackArgc < 1 {
		return sqerrors.New("the callback should expect a hook context as first argument")
	}
	if !reflect.TypeOf((*Context)(nil)).AssignableTo(callbackType.In(0)) {
		return sqerrors.New("the callback should expect a hook context as first argument")
	}
	// Check the argument count
	fnArgc := len(argTypes)
	if callbackArgc-1 != fnArgc && callbackArgc != fnArgc {
		return sqerrors.Errorf("the callback arguments count `%d` is not compatible to the hook arguments count `%d`", callbackArgc, fnArgc)
	}
	// Check arguments are assignable
	var i int
	for i = 1; i < callbackArgc; i++ {
		argPtrType := reflect.PtrTo(argTypes[i-1])
		callbackArgType := callbackType.In(i)
		if !argPtrType.AssignableTo(callbackArgType) {
			return sqerrors.Errorf("hook argument `%d` of type `%s` cannot be assigned to the callback argument `%d` of type `%s`", i-1, argPtrType, i, callbackArgType)
		}
	}
	return nil
}

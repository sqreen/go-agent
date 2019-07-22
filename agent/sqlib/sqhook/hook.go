// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Package sqhook provides a pure Go implementation of hooks to be inserted
// into function definitions in order to be able to attach at run time prolog
// and epilog callbacks getting read/write access to the arguments and returned
// values of the function call.
//
// A hook needs to be globally created and associated to a function symbol at
// package initialization time. Callbacks can then be accessed in the function
// call to pass the call arguments and return values.
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
//		type prolog = func(*A, *B, *C) (epilog, error)
// The expected epilog signature is:
//		type epilog = func(*R, *S, *T)
//
// Note 1: the prolog callback returns the epilog callback - which can be nil
// when not required - so that context can be shared using a closure.
//
// Note 2: a prolog for a method should accept the method receiver pointer as
// first argument wrapped into a `sqhook.MethodReceiver` value.
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
//				type Epilog = func(*[]byte, *error)
//				type Prolog = func(*int, *string) (Epilog, error)
//				// Get the prolog callback and call it if it is not nil
//				prolog := exampleHook.Prolog()
//				if prolog, ok := prolog.(Prolog); ok {
//					// Pass pointers to the arguments
//					epilog, err := prolog(&w, &r, &headers, &statusCode, &body)
//					// If an error is returned, the function execution is aborted.
//					// The epilog still needs to be called if set. A deferred call to it
//					// does the job.
//					if epilog != nil {
//						// Pass pointers to the return values
//						defer epilog(&ret1, &ret2)
//					}
//					if err != nil {
//						return
//					}
//				}
//			}
// 			/* .. function code ... */
//		}
//
//
// Main requirements:
//
// - Concurrent access and modification of callbacks.
// - Ability to read/write arguments and return values.
// - Hook to the prolog and epilog of a function.
// - Epilog callbacks should be able to recover from a panic.
// - Callbacks should be reentrant. If any context needs to be shared, it
//   should be done through the closure.
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
	// Currently attached callback.
	attached *PrologCallback
	// Symbol name where the hook is used. Required for the stringer.
	symbol string
}

// PrologCallback is an interface type to a prolog function.
// Given a function F:
// 		func F(A, B, C) (R, S, T)
// The expected prolog signature is:
//		type prolog = func(*A, *B, *C) (epilog, error)
// The expected epilog signature is:
//		type epilog = func(*R, *S, *T)
// The returned epilog value can be nil when there is no need for epilog.
type PrologCallback interface{}

// MethodReceiver should be the first argument of the prolog of a method.
type MethodReceiver struct{ Receiver interface{} }

// Errors that hooks can return in order to modify the control flow of the
// function.
type Error int

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
		panic("nil argument")
	}
	v := reflect.ValueOf(fn)
	fnType := v.Type()
	if fnType.Kind() != reflect.Func {
		panic("the argument is not a function type")
	}
	// If the symbol name cannot be retrieved
	symbol := runtime.FuncForPC(v.Pointer()).Name()
	if symbol == "" {
		panic("could not read the symbol name of the function")
	}
	// Create the hook, store it in the map and return it.
	hook := &Hook{
		symbol: symbol,
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

// Attach atomically attaches a prolog callback to the hook. It is
// possible to pass a `nil` value to remove the attached callback.
func (h *Hook) Attach(prolog PrologCallback) error {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&h.attached))
	if prolog == nil {
		atomic.StorePointer(addr, nil)
		return nil
	}
	if err := validateProlog(prolog, h.fnType); err != nil {
		return err
	}
	atomic.StorePointer(addr, unsafe.Pointer(&prolog))
	return nil
}

// Callbacks atomically accesses the attached prolog.
func (h *Hook) Prolog() (prolog PrologCallback) {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&h.attached))
	attached := (*PrologCallback)(atomic.LoadPointer(addr))
	if attached == nil {
		return nil
	}
	return *attached
}

// validateProlog validates that the prolog has the expected signature.
func validateProlog(prolog PrologCallback, fnType reflect.Type) (err error) {
	defer func() {
		if err != nil {
			err = sqerrors.Wrap(err, "prolog validation error")
		}
	}()
	// Create the list of argument types
	callbackArgsTypes := make([]reflect.Type, 0, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		callbackArgsTypes = append(callbackArgsTypes, fnType.In(i))
	}
	// Check the prolog args are pointers to the callback args
	prologType := reflect.TypeOf(prolog)
	if err := validateCallback(prologType, callbackArgsTypes); err != nil {
		return err
	}
	// Check the prolog returns two values
	if numPrologOut, numCallbackOut := prologType.NumOut(), 2; numPrologOut != numCallbackOut {
		return sqerrors.Errorf("wrong number of returned values, expected `%d` but got `%d`", numCallbackOut, numPrologOut)
	}
	// Check the second returned value is an error
	if ret1Type := prologType.Out(1); ret1Type != reflect.TypeOf((*error)(nil)).Elem() {
		return sqerrors.Errorf("unexpected second return type `%s` instead of `error`", ret1Type)
	}
	// Check the first returned value is the expected epilog type
	epilogType := prologType.Out(0)
	if err := validateEpilog(epilogType, fnType); err != nil {
		return err
	}
	return nil
}

// validateEpilog validates that the epilog has the expected signature.
func validateEpilog(epilogType reflect.Type, fnType reflect.Type) (err error) {
	defer func() {
		if err != nil {
			err = sqerrors.Wrap(err, "epilog validation error")
		}
	}()
	// Create the list of argument types
	callbackRetTypes := make([]reflect.Type, 0, fnType.NumOut())
	for i := 0; i < fnType.NumOut(); i++ {
		callbackRetTypes = append(callbackRetTypes, fnType.Out(i))
	}
	if err := validateCallback(epilogType, callbackRetTypes); err != nil {
		return err
	}
	if numOut := epilogType.NumOut(); numOut != 0 {
		return sqerrors.Errorf("unexpected number of return values `%d` instead of `0`", numOut)
	}
	return nil
}

// validateCallback validates the fact that the callback is a function whose
// first argument is the hook context and the rest of its arguments can be
// assigned the hook argument values.
func validateCallback(callbackType reflect.Type, argTypes []reflect.Type) error {
	// Check the callback is a function
	if callbackType.Kind() != reflect.Func {
		return sqerrors.New("the callback argument is not a function")
	}
	callbackArgc := callbackType.NumIn()
	// Check the callback accepts the same number of arguments than the function
	// Note that the method receiver is in the argument list of the type
	// definition.
	if callbackArgc != len(argTypes) {
		return sqerrors.Errorf("the callback should have the same arguments: `%d` callback arguments while expecting `%d`", callbackArgc, len(argTypes))
	}
	// Check arguments are pointers to the same types than the function arguments.
	if callbackArgc > 0 {
		i := 0
		if callbackType.In(0) == reflect.TypeOf(MethodReceiver{}) {
			i++
		}
		for ; i < callbackArgc; i++ {
			argPtrType := reflect.PtrTo(argTypes[i])
			callbackArgType := callbackType.In(i)
			if argPtrType != callbackArgType {
				return sqerrors.Errorf("argument `%d` has type `%s` instead of `%s`", i, callbackArgType, argPtrType)
			}
		}
	}
	return nil
}

func (h *Hook) String() string {
	return fmt.Sprintf("%s (%s)", h.symbol, h.fnType)
}

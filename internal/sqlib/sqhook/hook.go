// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

// Package sqhook allows to attach at run time (ie. hook) prolog and epilog
// callbacks to instrumented functions. They can read/write the function
// parameters and return values of function calls. Functions must have been
// instrumented using the instrumentation tool at
// github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool.
//
// Prolog callbacks get read/write access to the arguments of the function call
// before it gets executed, while epilog callbacks get read/write access to the
// return values before returning from the call. Therefore, the callback
// signatures depend on the function signature they attach to. The prolog
// callback type is checked when attached to a function hook.
//
// Given an instrumented function F:
// 		func F(A, B, C) (R, S, T)
// The expected prolog signature is:
//		type prolog = func(*A, *B, *C) (epilog, error)
// The expected epilog signature is:
//		type epilog = func(*R, *S, *T)
//
// Note: the prolog function returns the epilog function which can be nil
// when not required. Context from the prolog can be shared with the epilog
// using the epilog function closure.
//
// Main requirements
//
// - Concurrent access and modification of callbacks.
// - Ability to read/write arguments and return values.
// - Hook the prolog and epilog of a function.
// - Epilog callbacks should be able to recover a panic.
// - Callbacks should be reentrant. If any context needs to be shared, it
//   should be done through the epilog closure.
// - Fast callback calls thanks to regular pointer to function calls. No need
//   for any type-assertion.
// - Ability to rely on type assertions instead of using `reflect.Call()`.
//   The usage of dynamic calls using `reflect` is indeed much slower and
//   consumes memory.
// - Access and modification of callbacks need to be atomic.
//
package sqhook

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqgo"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook/internal"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"golang.org/x/xerrors"
)

//go:linkname _sqreen_instrumentation_descriptor _sqreen_instrumentation_descriptor
var _sqreen_instrumentation_descriptor *internal.InstrumentationDescriptorType

type symbolIndexType map[string]*Hook

// index of hooks by symbol string. The index is lazily created when symbols
// are searched. Note that due to the large amount of hooks, we avoid having
// a map of hook pointer in order to avoid GC overhead.
var index = make(symbolIndexType)

type Hook struct {
	// Symbol name of the function the hook is associated with.
	symbol string
	// Prolog function type expected by this hook.
	prologFuncType reflect.Type
	// Pointer to the prolog pointer. The value has type **prologFuncType, which
	// is checked at hook creation.
	prologVarAddr *unsafe.Pointer
}

// PrologCallback is an interface to a prolog function.
// Given a function F:
// 		func F(A, B, C) (R, S, T)
// The expected prolog signature is:
//		type prolog = func(*A, *B, *C) (epilog, error)
// The expected epilog signature is:
//		type epilog = func(*R, *S, *T)
// The returned epilog value can be nil when there is no need for epilog.
type (
	PrologCallback       interface{}
	PrologCallbackGetter interface {
		PrologCallback() PrologCallback
	}
	ReflectedPrologCallback = func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error)
	ReflectedEpilogCallback = func(results []reflect.Value)
)

// Errors that hooks can return in order to modify the control flow of the
// function.
type Error int

const (
	_ Error = iota
	// Abort the execution of the function by returning from it.
	AbortError
)

func (e Error) Error() string {
	switch e {
	case AbortError:
		return "abort function execution"
	default:
		return "unknown"
	}
}

// Static assertion that `Error` implements interface `error`
var _ error = Error(0)

func Health(expectedVersion string) error {
	if _sqreen_instrumentation_descriptor == nil || len(_sqreen_instrumentation_descriptor.HookTable) == 0 {
		return sqerrors.New("the program is not instrumented - please refer to docs.sqreen.com/go/installation in order to instrument your program")
	}

	if version := _sqreen_instrumentation_descriptor.Version; version != expectedVersion {
		return sqerrors.Errorf("the program is not properly instrumented: the agent and instrumentation tool versions must be the same - the tool version is `%s` while the agent version is `%s`", version, expectedVersion)
	}

	return nil
}

// Find returns the hook associated to the given symbol string when it was
// created using `New()`, nil otherwise.
func Find(symbol string) (*Hook, error) {
	return index.find(symbol)
}

// Try to find the `symbol` in the index first, otherwise try to load it from
// the hook table.
func (t symbolIndexType) find(symbol string) (*Hook, error) {
	// Lookup the symbol index first
	if hook, exists := index[symbol]; exists {
		return hook, nil
	}
	// Not found in the index: lookup the hook table
	return hookTableLookup(_sqreen_instrumentation_descriptor.HookTable, symbol, index)
}

func hookTableLookup(table internal.HookTableType, symbol string, index symbolIndexType) (found *Hook, err error) {
	id := normalizedHookID(symbol)
	// The API of sort.Search doesn't allow to abort, so we panic instead,
	// caught by sqsafe.Call.
	err = sqsafe.Call(func() error {
		sort.Search(len(table), func(i int) bool {
			entry := table[i]
			var descriptor internal.HookDescriptorType
			entry(&descriptor)
			hook, err := index.add(descriptor.Func, descriptor.PrologVar)
			if err != nil {
				panic(err) // abort
			}
			current := normalizedHookID(hook.symbol)
			cmp := strings.Compare(current, id)
			if cmp == 0 {
				found = hook
			}
			return cmp >= 0
		})
		return nil
	})

	var panicErr *sqsafe.PanicError
	if err != nil && xerrors.As(err, &panicErr) {
		return nil, sqerrors.Wrapf(panicErr.Err, "hook table lookup of symbol `%s`", symbol)
	}
	return found, nil
}

func normalizedHookID(symbol string) string {
	id := regexp.MustCompile(`[ *()]`).ReplaceAllString(symbol, "")
	return regexp.MustCompile(`[/.\-@]`).ReplaceAllString(id, "_")
}

// add creates the hook object for function `fn`, adds it to the find map and
// returns it. It returns an error if it is not possible.
func (t symbolIndexType) add(fn, prologVar interface{}) (h *Hook, err error) {
	// Check fn is a non-nil function value
	if fn == nil {
		return nil, sqerrors.New("unexpected function argument value `nil`")
	}
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if fnType.Kind() != reflect.Func {
		return nil, sqerrors.Errorf("unexpected function argument type: expecting a function value but got `%T`", fn)
	}

	// Get the symbol name
	symbol := runtime.FuncForPC(fnValue.Pointer()).Name()
	if symbol == "" {
		return nil, sqerrors.Errorf("could not read the symbol name of function `%T`", fn)
	}

	// Unvendor it so that it is not prefixed by `<app>/vendor/`
	symbol = sqgo.Unvendor(symbol)

	// Use the symbol name for better error messages
	defer func() {
		if err != nil {
			err = sqerrors.Wrapf(err, "symbol `%s`", symbol)
		}
	}()

	// The hook may have been already added by a previous lookup
	if hook, exists := t[symbol]; exists {
		return hook, nil
	}

	// Check the prolog variable is compatible with the function
	if prologVar == nil {
		return nil, sqerrors.New("unexpected prolog variable argument value `nil`")
	}
	prologVarValue := reflect.ValueOf(prologVar)
	prologFuncType := prologVarValue.Type()

	if err := validatePrologVar(fnType, prologFuncType); err != nil {
		return nil, sqerrors.Wrap(err, "prolog variable validation")
	}

	prologFuncType = prologFuncType.Elem().Elem()
	prologVarAddr := (*unsafe.Pointer)(unsafe.Pointer(prologVarValue.Pointer()))

	// Create the hook, store it in the map and return it.
	hook := &Hook{
		symbol:         symbol,
		prologFuncType: prologFuncType,
		prologVarAddr:  prologVarAddr,
	}
	t[symbol] = hook
	return hook, nil
}

func (h *Hook) String() string {
	return fmt.Sprintf("%s (%s)", h.symbol, h.prologFuncType)
}

// Attach atomically attaches a prolog function to the hook. The hook can be
// disabled with a `nil` prolog value.
func (h *Hook) Attach(prologs ...PrologCallback) error {
	addr := h.prologVarAddr
	if l := len(prologs); l == 0 || (l == 1 && prologs[0] == nil) {
		// Disable
		atomic.StorePointer(addr, nil)
		return nil
	}

	prologCallbacks := make([]PrologCallback, len(prologs))
	for i, prolog := range prologs {
		// Loop until the prolog type is not one of the above
	loop:
		for {
			switch actual := prolog.(type) {
			case ReflectedPrologCallback:
				prolog = makePrologCallback(h, actual)
			case PrologCallbackGetter:
				prolog = actual.PrologCallback()
			default:
				// Final type
				break loop
			}
		}

		if h.prologFuncType != reflect.TypeOf(prolog) {
			return sqerrors.Errorf("unexpected prolog type for hook `%s`: got `%T`, wanted `%s`", h, prolog, h.prologFuncType)
		}

		prologCallbacks[i] = prolog
	}

	// Create the prolog out of the prologCallbacks
	var prolog PrologCallback
	if l := len(prologCallbacks); l == 1 {
		prolog = prologCallbacks[0]
	} else {
		// Create a dynamic function calling the prolog
		prolog = makeMultiPrologCallback(h, prologCallbacks)
	}

	// Create a value having type "pointer to the prolog function"
	ptr := reflect.New(h.prologFuncType)
	// *ptr = prolog
	ptr.Elem().Set(reflect.ValueOf(prolog))
	// Atomically store it: *addr = ptr
	atomic.StorePointer(addr, unsafe.Pointer(ptr.Pointer()))
	return nil
}

func makeMultiPrologCallback(h *Hook, prologs []PrologCallback) PrologCallback {
	return makePrologCallback(h, func(params []reflect.Value) (epilog ReflectedEpilogCallback, err error) {
		safeCallErr := sqsafe.Call(func() error {
			epilogs := make([]reflect.Value, 0, len(prologs))
			defer func() {
				if len(epilogs) > 0 {
					epilog = func(results []reflect.Value) {
						for _, epilog := range epilogs {
							epilog.Call(results)
						}
					}
				}
			}()
			for _, prolog := range prologs {
				prologValue := reflect.ValueOf(prolog)
				results := prologValue.Call(params)
				if r0 := results[0]; !r0.IsNil() {
					epilogs = append(epilogs, r0)
				}
				if r1 := results[1]; !r1.IsNil() {
					err = r1.Interface().(error)
					return nil
				}
			}
			return nil
		})
		if safeCallErr != nil {
			// TODO: log this error once
		}
		return epilog, err
	})
}

func makePrologCallback(h *Hook, prolog ReflectedPrologCallback) PrologCallback {
	prologFuncType := h.prologFuncType
	epilogFuncType := h.prologFuncType.Out(0)
	return reflect.MakeFunc(prologFuncType, func(args []reflect.Value) (results []reflect.Value) {
		epilog, err := prolog(args)
		var epilogFuncValue reflect.Value
		if epilog != nil {
			epilogFuncValue = reflect.MakeFunc(epilogFuncType, func(args []reflect.Value) (results []reflect.Value) {
				epilog(args)
				return []reflect.Value{}
			})
		} else {
			epilogFuncValue = reflect.New(epilogFuncType).Elem()
		}
		// The error value is retrieved through its pointer because ValueOf returns
		// the concrete type value and error is an interface
		// cf. https://github.com/golang/go/issues/28761
		return []reflect.Value{epilogFuncValue, reflect.ValueOf(&err).Elem()}
	}).Interface()
}

// validatePrologVar validates that the prolog variable has the expected type.
// Given a function:
// 		func F(A, B, C) (R, S, T)
// The expected prolog variable to use is:
// 		var prologVarForF **prolog
func validatePrologVar(fnType, prologVarType reflect.Type) error {
	// Check the prolog variable type is a `**func`.
	if prologVarType.Kind() != reflect.Ptr ||
		prologVarType.Elem().Kind() != reflect.Ptr ||
		prologVarType.Elem().Elem().Kind() != reflect.Func {
		return sqerrors.Errorf("prolog variable type is not a `**func` but `%s`", prologVarType)
	}
	if err := validateProlog(fnType, prologVarType.Elem().Elem()); err != nil {
		return sqerrors.Wrap(err, "prolog function type validation")
	}
	return nil
}

func validateProlog(fnType reflect.Type, prologType reflect.Type) error {
	// Check the prolog is a function
	if prologType.Kind() != reflect.Func {
		return sqerrors.New("the prolog argument type is not a function")
	}
	// Create the list of expected argument types
	expectedArgs := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		expectedArgs[i] = fnType.In(i)
	}
	// Check the prolog args are pointers to the function args
	if err := validateCallbackArgs(prologType, expectedArgs); err != nil {
		return sqerrors.Wrap(err, "arguments validation")
	}
	// Check the prolog returns two values
	if numPrologOut, numCallbackOut := prologType.NumOut(), 2; numPrologOut != 2 {
		return sqerrors.Errorf("unexpected number result values: expected `%d` but got `%d`", numCallbackOut, numPrologOut)
	}
	// Check the second returned value is an error
	if retType := prologType.Out(1); retType != reflect.TypeOf((*error)(nil)).Elem() {
		return sqerrors.Errorf("unexpected second result value type `%s` instead of `error`", retType)
	}
	// Check the first returned value is the expected epilog type
	epilogType := prologType.Out(0)
	if err := validateEpilog(epilogType, fnType); err != nil {
		return sqerrors.Wrap(err, "epilog validation")
	}
	return nil
}

// validateEpilog validates that the epilog has the expected signature.
func validateEpilog(epilogType reflect.Type, fnType reflect.Type) error {
	// Check the epilog is a function
	if epilogType.Kind() != reflect.Func {
		return sqerrors.New("the epilog argument is not a function")
	}
	// Create the list of argument types
	callbackRetTypes := make([]reflect.Type, fnType.NumOut())
	for i := 0; i < fnType.NumOut(); i++ {
		callbackRetTypes[i] = fnType.Out(i)
	}
	// Check the epilog args are pointers to the function results
	if err := validateCallbackArgs(epilogType, callbackRetTypes); err != nil {
		return sqerrors.Wrap(err, "arguments validation")
	}
	// Check the prolog doesn't return values
	if numOut, expectedOut := epilogType.NumOut(), 0; numOut != expectedOut {
		return sqerrors.Errorf("unexpected number of return values `%d` instead of `%d`", numOut, expectedOut)
	}
	return nil
}

// validateCallbackArgs validates that the callback arguments are pointer to
// the given argument types.
func validateCallbackArgs(callbackType reflect.Type, expectedArgs []reflect.Type) error {
	// Check the callback has the same number of arguments than the function.
	// Note that the method receiver is also in the argument list.
	callbackArgc := callbackType.NumIn()
	if expectedArgc := len(expectedArgs); callbackArgc != expectedArgc {
		return sqerrors.Errorf("unexpected number of arguments: got `%d` instead of `%d`", callbackArgc, expectedArgc)
	}

	if callbackArgc == 0 {
		return nil
	}

	// Check arguments are pointers to the same types than the function arguments.
	// The first argument is the only exception which may be a method receiver.
	//i := 0
	//if callbackType.In(i) == reflect.TypeOf((*MethodReceiver)(nil)).Elem() {
	//	i++
	//}
	for i := 0; i < callbackArgc; i++ {
		expectedArgType := reflect.PtrTo(expectedArgs[i])
		callbackArgType := callbackType.In(i)
		if expectedArgType != callbackArgType {
			return sqerrors.Errorf("argument `%d` has type `%s` instead of `%s`", i, callbackArgType, expectedArgType)
		}
	}
	return nil
}

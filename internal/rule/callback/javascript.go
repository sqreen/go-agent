// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"context"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/robertkrimen/otto"
	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type GenericCallbackConfig interface {
	Config
	BindingAccessors() []string
	JSCallbacks(string) string
}

func NewJSExecCallback(rule RuleFace, prologFuncType reflect.Type) (sqhook.ReflectedPrologCallback, error) {
	cfg, ok := rule.Config().(GenericCallbackConfig)
	if !ok {
		return nil, sqerrors.Errorf("unexpected configuration type `%T`", rule.Config())
	}

	// todo: newVMPool()
	vm := otto.New()
	{
		if src := cfg.JSCallbacks("pre"); src != "" {
			_, err := vm.Run(src)
			if err != nil {
				return nil, sqerrors.New("could not ")
			}
			if v, err := vm.Run("pre"); err != nil || !v.IsObject() {
				return nil, sqerrors.New("could not get the pre object value")
			}
		}

		if src := cfg.JSCallbacks("post"); src != "" {
			_, err := vm.Run(src)
			if err != nil {
				return nil, sqerrors.New("could not ")
			}
			if v, err := vm.Run("post"); err != nil || !v.IsObject() {
				return nil, sqerrors.New("could not post object value")
			}
		}
	}
	pool := sync.Pool{
		New: func() interface{} {
			return vm.Copy()
		},
	}

	bindingAccessors := cfg.BindingAccessors()
	args := make([]bindingaccessor.BindingAccessorFunc, len(bindingAccessors))
	for i, expr := range bindingAccessors {
		ba, err := bindingaccessor.Compile(expr)
		if err != nil {
			return nil, sqerrors.Wrapf(err, "binding accessor compilation of argument %d", i)
		}
		args[i] = ba
	}

	strategy := cfg.Strategy()

	return func(params []reflect.Value) (epilogFunc sqhook.ReflectedEpilogCallback, err error) {
		// sqsafe call + logerror
		goCtx := params[strategy.Protection.Context.ArgIndex].Elem().Interface().(context.Context)
		ctx := httpprotection.FromContext(goCtx)

		// Make benefit from the fact this is a protection callback to also get the request reader
		baCtx, err := NewCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, params, nil, ctx.RequestReader)
		if err != nil {
			return nil, nil
		}
		jsArgs := make([]interface{}, len(args))
		for i, a := range args {
			v, err := a(baCtx)
			if err != nil {
				panic(err)
			}
			if v == nil {
				v = struct{}{}
			}
			jsArgs[i] = v
		}

		vm := pool.Get().(*otto.Otto)
		defer pool.Put(vm)
		// TODO: how to use Object?
		pre, err := vm.Run("pre")
		if err != nil {
			panic(err)
		}

		r, err := pre.Call(otto.NullValue(), jsArgs...)
		if err != nil {
			panic(err)
		}

		if !r.IsObject() {
			return nil, nil
		}
		exported, err := r.Export()
		if err != nil {
			panic(err)
		}

		result := exported.(map[string]interface{})
		var raise bool
		if status, exists := result["status"]; exists && status.(string) == "raise" {
			raise = true
		}
		if !raise {
			return nil, nil
		}

		blocking := cfg.BlockingMode()
		metadata := result["record"].(map[string]interface{})
		abortErr := abortError{}
		st := sqerrors.StackTrace(errors.WithStack(abortErr))
		ctx.AddAttackEvent(rule.NewAttackEvent(blocking, noScrub(metadata), st))
		if !blocking {
			return nil, nil
		}

		defer ctx.CancelHandlerContext()
		ctx.WriteDefaultBlockingResponse()
		return func(results []reflect.Value) {
			errorIndex := strategy.Protection.BlockStrategy.RetIndex
			results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
		}, sqhook.AbortError
	}, nil
}

type noScrub map[string]interface{}

func (n noScrub) NoScrub() {}

type abortError struct{}

func (abortError) Error() string { return "abort" }

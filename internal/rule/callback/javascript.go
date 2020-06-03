// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"reflect"
	"sync"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
)

func NewJSExecCallback(rule RuleFace, cfg ReflectedCallbackConfig) (sqhook.ReflectedPrologCallback, error) {
	pool := newVMPool(cfg)
	sqassert.NotNil(pool)
	strategy := cfg.Strategy()
	sqassert.NotNil(strategy)

	return func(params []reflect.Value) (epilogFunc sqhook.ReflectedEpilogCallback, prologErr error) {
		err := sqsafe.Call(func() error {
			ctx := getProtectionContext(strategy.Protection, params)
			if ctx == nil {
				return nil
			}

			vm := pool.get()
			defer pool.put(vm)

			// Make benefit from the fact this is a protection callback to also get the request reader
			baCtx, err := NewCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, params, nil, ctx.RequestReader)
			if err != nil {
				return err
			}

			// TODO: post callback as soon as it's needed by a protection
			//if vm.hasPost() {
			//}

			if vm.hasPre() {
				result, err := vm.callPre(baCtx)
				if err != nil {
					// TODO: api adding more information to the error such as the
					//   rule name, etc.
					ctx.Logger().Error(err)
					return nil
				}

				raise := result.Status == "raise"
				if !raise {
					return nil
				}

				// Create the attack event
				blocking := cfg.BlockingMode()
				metadata := result.Record
				abortErr := abortError{}
				st := sqerrors.StackTrace(errors.WithStack(abortErr))
				ctx.AddAttackEvent(rule.NewAttackEvent(blocking, noScrub(metadata), st))

				// If not in blocking mode, return here and don't block the request
				if !blocking {
					return nil
				}

				// Abort the request handler context
				defer ctx.CancelHandlerContext()

				// Write the blocking response
				ctx.WriteDefaultBlockingResponse()

				// Abort the function call according to the blocking strategy
				epilogFunc = func(results []reflect.Value) {
					errorIndex := strategy.Protection.BlockStrategy.RetIndex
					results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
				}
				prologErr = sqhook.AbortError
			}

			return nil
		})
		if err != nil {
			// TODO: log this error
		}
		return
	}, nil
}

type vmPool sync.Pool

type runtime struct {
	vm        *goja.Runtime
	pre, post *jsCallbackFunc
}

type jsCallbackFunc struct {
	callback       goja.Callable
	funcCallParams []bindingaccessor.BindingAccessorFunc
}

func newVMPool(cfg ReflectedCallbackConfig) *vmPool {
	preFuncDecl, preFuncCallParams := cfg.Pre()
	postFuncDecl, postFuncCallParams := cfg.Post()
	sqassert.True(preFuncDecl != nil || postFuncDecl != nil)

	return (*vmPool)(&sync.Pool{
		New: func() interface{} {
			var pre, post goja.Callable

			vm := goja.New()
			vm.SetFieldNameMapper(goja.TagFieldNameMapper("goja", false))

			if preFuncDecl != nil {
				_, err := vm.RunProgram(preFuncDecl)
				sqassert.NoError(err)
				if err := vm.ExportTo(vm.Get("pre"), &pre); err != nil {
					return sqerrors.Wrap(err, "retrieving `pre` function")
				}
			}

			if postFuncDecl != nil {
				v, err := vm.RunProgram(postFuncDecl)
				sqassert.NoError(err)
				//if err := vm.ExportTo(r.vm.Get("pre"), &pre); err != nil {
				if err := vm.ExportTo(v, &post); err != nil {
					return sqerrors.Wrap(err, "retrieving `post` function")
				}
			}

			return &runtime{
				vm: vm,
				pre: &jsCallbackFunc{
					callback:       pre,
					funcCallParams: preFuncCallParams,
				},
				post: &jsCallbackFunc{
					callback:       post,
					funcCallParams: postFuncCallParams,
				},
			}
		},
	})
}

func (vm *vmPool) unwrap() *sync.Pool {
	return (*sync.Pool)(vm)
}

func (vm *vmPool) get() *runtime {
	return vm.unwrap().Get().(*runtime)
}

func (vm *vmPool) put(r *runtime) {
	vm.unwrap().Put(r)
}

func (r *runtime) hasPre() bool {
	return r.pre != nil
}

func (r *runtime) hasPost() bool {
	return r.post != nil
}

type jsCallbackResult struct {
	Status string                 `goja:"status"`
	Record map[string]interface{} `goja:"record"`
}

func (r *runtime) callPre(baCtx bindingaccessor.Context) (*jsCallbackResult, error) {
	sqassert.True(r.hasPre())
	result := &jsCallbackResult{}

	if err := call(r.vm, r.pre, baCtx, result); err != nil {
		return nil, err
	}

	return result, nil
}

func call(vm *goja.Runtime, descr *jsCallbackFunc, baCtx bindingaccessor.Context, result interface{}) error {
	jsParams := make([]goja.Value, len(descr.funcCallParams))
	for i, ba := range descr.funcCallParams {
		v, err := ba(baCtx)
		if err != nil {
			return err
		}

		var jsVal goja.Value
		if v == nil {
			jsVal = vm.NewObject()
		} else {
			jsVal = vm.ToValue(v)
		}

		jsParams[i] = jsVal
	}

	v, err := descr.callback(goja.Undefined(), jsParams...)
	if err != nil {
		return err
	}

	return vm.ExportTo(v, result)
}

func getProtectionContext(protection *api.ReflectedCallbackProtectionConfig, params []reflect.Value) *httpprotection.RequestContext {
	return httpprotection.FromGLS()
}

type noScrub map[string]interface{}

func (n noScrub) NoScrub() {}

type abortError struct{}

func (abortError) Error() string { return "abort" }

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"reflect"
	"sync"

	"github.com/dop251/goja"
	"github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/sqreen/go-agent/sdk/types"
)

func NewJSExecCallback(r RuleContext, cfg JSReflectedCallbackConfig) (sqhook.ReflectedPrologCallback, error) {
	pool := newVMPool(cfg)
	// TODO: move this into a JSNativeRuleContext
	sqassert.NotNil(pool)
	strategy := cfg.Strategy()
	sqassert.NotNil(strategy)

	return func(params []reflect.Value) (epilogFunc sqhook.ReflectedEpilogCallback, prologErr error) {
		vm := pool.get()
		defer pool.put(vm)

		var blocked bool

		if vm.hasPre() {
			r.Pre(func(c CallbackContext) {
				baCtx, err := NewReflectedCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, c.ProtectionContext(), params, nil, cfg.Data())
				if err != nil {
					c.Logger().Error(err)
					return
				}

				result, err := vm.callPre(baCtx)
				if err != nil {
					// TODO: api adding more information to the error such as the
					//   rule name, etc.
					c.Logger().Error(err)
					return
				}

				if raise := result.Status == "raise"; !raise {
					return
				}

				blocked = c.HandleAttack(true, noScrub(result.Record))
				if !blocked {
					return
				}

				// Abort the function call according to the blocking strategy
				epilogFunc = func(results []reflect.Value) {
					abortErr := types.SqreenError{Err: attackError{}}
					errorIndex := strategy.Protection.BlockStrategy.RetIndex
					results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
				}
				prologErr = sqhook.AbortError
			})
		}

		if blocked {
			return
		}

		if vm.hasPost() {
			epilogFunc = func(results []reflect.Value) {
				r.Post(func(c CallbackContext) {
					baCtx, err := NewReflectedCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, c.ProtectionContext(), params, results, cfg.Data())
					if err != nil {
						c.Logger().Error(err)
						return
					}

					result, err := vm.callPost(baCtx)
					if err != nil {
						// TODO: api adding more information to the error such as the
						//   rule name, etc.
						c.Logger().Error(err)
						return
					}

					if raise := result.Status == "raise"; !raise {
						return
					}

					blocked = c.HandleAttack(true, noScrub(result.Record))
					if !blocked {
						return
					}

					// Abort the function call according to the blocking strategy
					abortErr := types.SqreenError{Err: attackError{}}
					errorIndex := strategy.Protection.BlockStrategy.RetIndex
					results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
				})
			}
			prologErr = nil
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

type fileNameMapper struct {
	goja.FieldNameMapper
}

func (m fileNameMapper) FieldName(t reflect.Type, f reflect.StructField) string {
	if n := m.FieldNameMapper.FieldName(t, f); n != "" {
		return n
	}
	return f.Name
}

func newVMPool(cfg JSReflectedCallbackConfig) *vmPool {
	preFuncDecl, preFuncCallParams := cfg.Pre()
	postFuncDecl, postFuncCallParams := cfg.Post()
	sqassert.True(preFuncDecl != nil || postFuncDecl != nil)

	return (*vmPool)(&sync.Pool{
		New: func() interface{} {
			vm := goja.New()
			vm.SetFieldNameMapper(fileNameMapper{goja.TagFieldNameMapper("goja", false)})

			var pre, post *jsCallbackFunc

			if preFuncDecl != nil {
				_, err := vm.RunProgram(preFuncDecl)
				sqassert.NoError(err)
				var fn goja.Callable
				if err := vm.ExportTo(vm.Get("pre"), &fn); err != nil {
					return sqerrors.Wrap(err, "retrieving `pre` function")
				}

				pre = &jsCallbackFunc{
					callback:       fn,
					funcCallParams: preFuncCallParams,
				}
			}

			if postFuncDecl != nil {
				_, err := vm.RunProgram(postFuncDecl)
				sqassert.NoError(err)
				var fn goja.Callable
				if err := vm.ExportTo(vm.Get("post"), &fn); err != nil {
					return sqerrors.Wrap(err, "retrieving `post` function")
				}

				post = &jsCallbackFunc{
					callback:       fn,
					funcCallParams: postFuncCallParams,
				}
			}

			return &runtime{
				vm:   vm,
				pre:  pre,
				post: post,
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

func (r *runtime) callPost(baCtx bindingaccessor.Context) (*jsCallbackResult, error) {
	sqassert.True(r.hasPost())
	result := &jsCallbackResult{}
	if err := call(r.vm, r.post, baCtx, result); err != nil {
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

type noScrub map[string]interface{}

func (n noScrub) NoScrub() {}

type attackError struct{}

func (attackError) Error() string { return "attack detected" }

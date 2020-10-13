// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/binding-accessor"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	sdk_types "github.com/sqreen/go-agent/sdk/types"
	"github.com/sqreen/go-libsqreen/waf"
	waf_types "github.com/sqreen/go-libsqreen/waf/types"
)

const defaultMaxWAFTimeBudget = 3 * time.Millisecond

var (
	WAFTimeoutLogOnce,
	WAFErrorLogOnce,
	FunctionWAFPreErrorLogOnce,
	FunctionWAFPostErrorLogOnce sync.Once
)

type wafCallbackObject struct {
	wafRule waf_types.Rule
	prolog  sqhook.PrologCallback
}

func (o *wafCallbackObject) PrologCallback() sqhook.PrologCallback {
	return o.prolog
}

func (o *wafCallbackObject) Close() error {
	return o.wafRule.Close()
}

func prepareWAF(cfg NativeCallbackConfig) (waf_types.Rule, map[string]bindingaccessor.BindingAccessorFunc, time.Duration, error) {
	data, ok := cfg.Data().(*api.WAFRuleDataEntry)
	if !ok {
		return nil, nil, 0, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", cfg.Data(), data)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, nil, 0, sqerrors.New("could not generate a uuid")
	}

	wafRule, err := waf.NewRule(id.String(), data.WAFRules, bindingaccessor.NewValueMaxElements, bindingaccessor.MaxExecutionDepth)
	if err != nil {
		return nil, nil, 0, sqerrors.Wrap(err, "could not instantiate the in-app waf rule")
	}

	if len(data.BindingAccessors) == 0 {
		return nil, nil, 0, sqerrors.New("unexpected empty list of binding accessors")
	}
	bindingAccessors := make(map[string]bindingaccessor.BindingAccessorFunc, len(data.BindingAccessors))
	for _, expr := range data.BindingAccessors {
		ba, err := bindingaccessor.Compile(expr)
		if err != nil {
			return nil, nil, 0, sqerrors.Wrapf(err, "could not compile binding accessor expression `%s`", expr)
		}
		bindingAccessors[expr] = ba
	}

	var timeout time.Duration
	if data.Timeout != 0 {
		timeout = time.Duration(data.Timeout) * time.Millisecond
	} else {
		timeout = defaultMaxWAFTimeBudget
	}

	return wafRule, bindingAccessors, timeout, nil
}

func NewWAFCallback(rule RuleContext, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	wafRule, bindingAccessors, timeout, err := prepareWAF(cfg)
	if err != nil {
		return nil, sqerrors.Wrap(err, "unexpected configuration error")
	}
	return newWAFPrologCallback(rule, cfg.BlockingMode(), wafRule, bindingAccessors, timeout), nil
}

type (
	WAFPrologCallbackType = http_protection.WAFPrologCallbackType
	WAFEpilogCallbackType = http_protection.WAFEpilogCallbackType
)

var ErrWAFProtection = errors.New("waf protection triggered")

func newWAFPrologCallback(rule RuleContext, blockingMode bool, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration) *wafCallbackObject {
	return &wafCallbackObject{
		wafRule: wafRule,
		prolog:  makeWAFPrologCallback(rule, blockingMode, wafRule, bindingAccessors, timeout),
	}
}

func runWAF(c CallbackContext, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, wafRule waf_types.Rule, timeout time.Duration) (blocked bool, err error) {
	baCtx, err := MakeWAFCallbackBindingAccessorContext(c)
	if err != nil {
		return false, err
	}

	args := make(waf_types.DataSet, len(bindingAccessors))
	for expr, ba := range bindingAccessors {
		value, err := ba(baCtx)
		if err != nil {
			// Log the error and continue
			c.Logger().Error(sqerrors.Wrapf(err, "binding accessor execution error `%s`", expr))
			continue
		}
		if value == nil {
			// Skip unset values
			continue
		}
		args[expr] = value
	}

	// TODO: args caching
	action, info, err := wafRule.Run(args, timeout)
	if err != nil {
		if err == waf_types.ErrTimeout {
			WAFTimeoutLogOnce.Do(func() {
				c.Logger().Error(sqerrors.New("WAF timeout"))
			})
			return false, nil
		}
		return false, newWAFRunError(err, args, timeout)
	}

	if action == waf_types.NoAction {
		return false, nil
	}

	attackInfo := api.WAFAttackInfo{WAFData: info}

	return c.HandleAttack(action == waf_types.BlockAction, attackInfo), nil
}

func makeWAFPrologCallback(rule RuleContext, blockingMode bool, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration) sqhook.PrologCallback {
	return func(**http_protection.ProtectionContext) (epilog http_protection.BlockingEpilogCallbackType, prologErr error) {
		rule.Pre(func(c CallbackContext) {
			blocked, err := runWAF(c, bindingAccessors, wafRule, timeout)
			if err != nil {
				WAFErrorLogOnce.Do(func() {
					c.Logger().Error(sqerrors.Wrap(err, "WAF execution error"))
				})
			}

			if !blocked {
				epilog, prologErr = nil, nil
				return
			}

			// Return the epilog and abort the call.
			epilog, prologErr = func(err *error) {
				// An error needs to be written in order to abort handling the
				// request.
				*err = sdk_types.SqreenError{Err: ErrWAFProtection}
			}, sqhook.AbortError
		})
		return
	}
}

type wafRunErrorInfo struct {
	Input   waf_types.DataSet
	Timeout time.Duration
}

func newWAFRunError(err error, args waf_types.DataSet, timeout time.Duration) error {
	return sqerrors.WithInfo(err, wafRunErrorInfo{
		Input:   args,
		Timeout: timeout,
	})
}

func NewFunctionWAFCallback(r RuleContext, cfg ReflectedCallbackConfig, functionWAFCfg *api.RuleFunctionWAFCallbacks) (sqhook.PrologCallback, error) {
	if functionWAFCfg == nil {
		return nil, sqerrors.New("unexpected nil function waf configuration")
	}
	if len(functionWAFCfg.Pre) == 0 && len(functionWAFCfg.Post) == 0 {
		return nil, sqerrors.New("unexpected empty pre and post list of binding accessors")
	}

	wafRule, bindingAccessors, timeout, err := prepareWAF(cfg)
	if err != nil {
		return nil, sqerrors.Wrap(err, "unexpected configuration error")
	}

	pre, err := compileFunctionWAFBindingAccessors(functionWAFCfg.Pre)
	if err != nil {
		return nil, sqerrors.Wrap(err, "could not compile the pre binding accessors")
	}

	post, err := compileFunctionWAFBindingAccessors(functionWAFCfg.Post)
	if err != nil {
		return nil, sqerrors.Wrap(err, "could not compile the post binding accessors")
	}

	return newFunctionWAFPrologCallback(r, wafRule, bindingAccessors, timeout, cfg.Strategy(), pre, post), nil
}

type functionWAFBindingAccessorMap map[*bindingaccessor.BindingAccessorFunc]bindingaccessor.BindingAccessorFunc

func compileFunctionWAFBindingAccessors(bas map[string]string) (functionWAFBindingAccessorMap, error) {
	if len(bas) == 0 {
		return nil, nil
	}

	r := make(functionWAFBindingAccessorMap, len(bas))
	for k, v := range bas {
		kf, err := bindingaccessor.Compile(k)
		if err != nil {
			return nil, sqerrors.Wrapf(err, "could not compile the binding accessor expression `%s`", k)
		}

		vf, err := bindingaccessor.Compile(v)
		if err != nil {
			return nil, sqerrors.Wrapf(err, "could not compile the binding accessor expression `%s`", v)
		}

		r[&kf] = vf
	}
	return r, nil
}

func newFunctionWAFPrologCallback(r RuleContext, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration, strategy *api.ReflectedCallbackConfig, pre, post functionWAFBindingAccessorMap) *wafCallbackObject {
	return &wafCallbackObject{
		wafRule: wafRule,
		prolog:  makeFunctionWAFPrologCallback(r, wafRule, bindingAccessors, timeout, strategy, pre, post),
	}
}

func makeFunctionWAFPrologCallback(r RuleContext, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration, strategy *api.ReflectedCallbackConfig, pre, post functionWAFBindingAccessorMap) sqhook.ReflectedPrologCallback {
	sqassert.True(len(pre) > 0 || len(post) > 0)

	return func(params []reflect.Value) (epilog sqhook.ReflectedEpilogCallback, prologErr error) {
		if l := len(pre); l > 0 {
			var preErr error
			r.Pre(func(c CallbackContext) {
				blocked, err := runFunctionWAF(c, bindingAccessors, wafRule, timeout, pre, strategy, params, nil)
				if err != nil {
					FunctionWAFPreErrorLogOnce.Do(func() {
						c.Logger().Error(sqerrors.Wrap(err, "function waf error: pre"))
					})
					preErr = err
					return
				}

				if blocked {
					epilog = func(results []reflect.Value) {
						errorIndex := strategy.Protection.BlockStrategy.RetIndex
						abortErr := sdk_types.SqreenError{Err: ErrWAFProtection}
						results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
					}
					prologErr = sqhook.AbortError
					return
				}
			})
			if preErr != nil {
				return nil, nil
			}
		}

		if l := len(post); l > 0 {
			epilog = func(results []reflect.Value) {
				r.Post(func(c CallbackContext) {
					blocked, err := runFunctionWAF(c, bindingAccessors, wafRule, timeout, post, strategy, params, results)
					if err != nil {
						FunctionWAFPreErrorLogOnce.Do(func() {
							c.Logger().Error(sqerrors.Wrap(err, "function waf error: post"))
						})
						return
					}

					if !blocked {
						return
					}

					errorIndex := strategy.Protection.BlockStrategy.RetIndex
					abortErr := sdk_types.SqreenError{Err: ErrWAFProtection}
					results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
				})
			}
		}

		return
	}
}

func runFunctionWAF(c CallbackContext, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, wafRule waf_types.Rule, timeout time.Duration, extraParamAccessors functionWAFBindingAccessorMap, strategy *api.ReflectedCallbackConfig, params []reflect.Value, results []reflect.Value) (blocked bool, err error) {
	baCtx, err := NewReflectedCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, c, params, results, nil)
	if err != nil {
		err = sqerrors.Wrap(err, "unexpected error while creating the binding accessor context")
		return false, err
	}

	hasExtraParams := false
	for kf, vf := range extraParamAccessors {
		sqassert.NotNil(kf)
		v, err := (*kf)(baCtx)
		if err != nil {
			return false, sqerrors.Wrap(err, "unexpected error while executing the extra param name binding accessor")
		}

		name, ok := v.(string)
		if !ok {
			return false, sqerrors.Wrapf(err, "unexpected extra param name type: got `%T` instead of `%T`", v, name)
		}

		v, err = vf(baCtx)
		if err != nil {
			return false, sqerrors.Wrap(err, "unexpected error while executing the extra param value binding accessor")
		}

		// Skip nil values
		if v == nil {
			continue
		}

		// TODO: move this in a separate callback so that it doesn't depend on the WAF
		//   rule and can be always enabled
		c.ProtectionContext().AddRequestParam(name, v)

		hasExtraParams = true
	}

	// Do not run the WAF when no new request params were added
	if !hasExtraParams {
		return false, nil
	}

	return runWAF(c, bindingAccessors, wafRule, timeout)
}

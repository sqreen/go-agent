// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"errors"
	"io"
	"reflect"
	"time"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/event"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/span"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	sdk_types "github.com/sqreen/go-agent/sdk/types"
	"github.com/sqreen/go-libsqreen/waf"
	waf_types "github.com/sqreen/go-libsqreen/waf/types"
)

const defaultMaxWAFTimeBudget = 3 * time.Millisecond

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

type reactiveWAFCloser struct {
	io.Closer
	span.EventListener
}

func prepareWAF(cfg NativeCallbackConfig) (waf_types.Rule, map[string]bindingaccessor.BindingAccessorFunc, time.Duration, error) {
	data, ok := cfg.Data().(*api.BindingAccessorWAFRuleDataEntry)
	if !ok {
		return nil, nil, 0, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", cfg.Data(), data)
	}

	// TODO: align the binding accessor max depth/lengths values with the WAF
	// encoder settings
	wafRule, err := waf.NewRule(data.WAFRules)
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
	return newWAFPrologCallback(rule, wafRule, bindingAccessors, timeout), nil
}

type (
	WAFPrologCallbackType = http_protection.WAFPrologCallbackType
	WAFEpilogCallbackType = http_protection.WAFEpilogCallbackType
)

var ErrWAFProtection = errors.New("waf protection triggered")

func newWAFPrologCallback(rule RuleContext, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration) *wafCallbackObject {
	return &wafCallbackObject{
		wafRule: wafRule,
		prolog:  makeWAFPrologCallback(rule, wafRule, bindingAccessors, timeout),
	}
}

func runBindingAccessorWAF(c CallbackContext, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, wafRule waf_types.Rule, timeout time.Duration) (blocked bool, err error) {
	p := c.ProtectionContext()

	// Check that we have `timeout` amount of time available
	if p.DeadlineExceeded(timeout) {
		return false, nil
	}

	baCtx, err := NewRequestBindingAccessorContext(p)
	if err != nil {
		type errKey struct{}
		return false, sqerrors.WithKey(err, errKey{})
	}

	args := make(waf_types.DataSet, len(bindingAccessors))
	for expr, ba := range bindingAccessors {
		value, err := ba(baCtx)
		if err != nil {
			// Log the error and continue
			type errKey string
			c.Logger().Error(sqerrors.WithKey(sqerrors.Wrapf(err, "binding accessor execution error `%s`", expr), errKey(expr)))
			continue
		}
		if value == nil {
			// Skip unset values
			continue
		}
		args[expr] = value

		// Check we haven't exceeded the time deadline if any
		if p.DeadlineExceeded(0) {
			return false, nil
		}
	}

	return runWAF(c, wafRule, args, timeout)
}

func makeWAFPrologCallback(rule RuleContext, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration) sqhook.PrologCallback {
	return func(**http_protection.ProtectionContext) (epilog http_protection.BlockingEpilogCallbackType, prologErr error) {
		rule.Pre(func(c CallbackContext) error {
			blocked, err := runBindingAccessorWAF(c, bindingAccessors, wafRule, timeout)
			if err != nil {
				return sqerrors.Wrap(err, "WAF execution error")
			}

			if !blocked {
				epilog, prologErr = nil, nil
				return nil
			}

			// Return the epilog and abort the call.
			epilog, prologErr = func(err *error) {
				// An error needs to be written in order to abort handling the
				// request.
				*err = sdk_types.SqreenError{Err: ErrWAFProtection}
			}, sqhook.AbortError
			return nil
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
			r.Pre(func(c CallbackContext) (err error) {
				defer func() {
					preErr = err
				}()

				blocked, err := runFunctionWAF(c, bindingAccessors, wafRule, timeout, pre, strategy, params, nil)
				if err != nil {
					return sqerrors.Wrap(err, "function waf error: pre")
				}

				if blocked {
					epilog = func(results []reflect.Value) {
						errorIndex := strategy.Protection.BlockStrategy.RetIndex
						abortErr := sdk_types.SqreenError{Err: ErrWAFProtection}
						results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
					}
					prologErr = sqhook.AbortError
				}
				return nil
			})
			if preErr != nil {
				return nil, nil
			}
		}

		if l := len(post); l > 0 {
			epilog = func(results []reflect.Value) {
				r.Post(func(c CallbackContext) error {
					blocked, err := runFunctionWAF(c, bindingAccessors, wafRule, timeout, post, strategy, params, results)
					if err != nil {
						type keyErr struct{}
						return sqerrors.WithKey(sqerrors.Wrap(err, "function waf error: post"), keyErr{})
					}

					if !blocked {
						return nil
					}

					errorIndex := strategy.Protection.BlockStrategy.RetIndex
					abortErr := sdk_types.SqreenError{Err: ErrWAFProtection}
					results[errorIndex].Elem().Set(reflect.ValueOf(abortErr))
					return nil
				})
			}
		}

		return
	}
}

func runFunctionWAF(c CallbackContext, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, wafRule waf_types.Rule, timeout time.Duration, extraParamAccessors functionWAFBindingAccessorMap, strategy *api.ReflectedCallbackConfig, params []reflect.Value, results []reflect.Value) (blocked bool, err error) {
	baCtx, err := NewReflectedCallbackBindingAccessorContext(strategy.BindingAccessor.Capabilities, c.ProtectionContext(), params, results, nil)
	if err != nil {
		type errKey struct{}
		err = sqerrors.Wrap(err, "unexpected error while creating the binding accessor context")
		err = sqerrors.WithKey(err, errKey{})
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

		hasExtraParams = true
	}

	// Do not run the WAF when no new request params were added
	if !hasExtraParams {
		return false, nil
	}

	return runBindingAccessorWAF(c, bindingAccessors, wafRule, timeout)
}

func NewReactiveWAFCallback(rule RuleContext, cfg NativeCallbackConfig) (span.EventListener, error) {
	wafRule, subscriptions, timeout, err := prepareReactiveWAF(cfg)
	if err != nil {
		return nil, sqerrors.Wrap(err, "unexpected configuration error")
	}
	return reactiveWAFCloser{
		Closer:        wafRule,
		EventListener: newReactiveWAFCallback(rule, wafRule, subscriptions, timeout),
	}, nil
}

func prepareReactiveWAF(cfg NativeCallbackConfig) (waf_types.Rule, []map[string]struct{}, time.Duration, error) {
	data, ok := cfg.Data().(*api.ReactiveWAFRuleDataEntry)
	if !ok {
		return nil, nil, 0, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `%T`", cfg.Data(), data)
	}

	if len(data.Subscriptions) == 0 {
		return nil, nil, 0, sqerrors.New("unexpected empty list of subscriptions")
	}

	wafRule, err := waf.NewRule(data.WAFRules)
	if err != nil {
		return nil, nil, 0, sqerrors.Wrap(err, "could not instantiate the in-app waf rule")
	}

	var timeout time.Duration
	if data.Timeout != 0 {
		timeout = time.Duration(data.Timeout) * time.Millisecond
	} else {
		timeout = defaultMaxWAFTimeBudget
	}

	subscriptions := make([]map[string]struct{}, len(data.Subscriptions))
	for i, addresses := range data.Subscriptions {
		set := make(map[string]struct{}, len(addresses))
		for _, address := range addresses {
			set[address] = struct{}{}
		}
		subscriptions[i] = set
	}

	return wafRule, subscriptions, timeout, nil
}

func newReactiveWAFCallback(rule RuleContext, wafRule waf_types.Rule, subscriptions []map[string]struct{}, timeout time.Duration) span.EventListener {
	return span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		wafCtx := waf.NewAdditiveContext(wafRule)
		if wafCtx == nil {
			return nil
		}

		s.OnEnd(func(results span.AttributeGetter) error {
			wafCtx.Close()
			wafCtx = nil
			return nil
		})

		runWAF := func(s span.AttributeGetter) (spanErr error) {
			rule.Pre(func(c CallbackContext) error {
				blocked, err := runReactiveWAF(s, c, wafCtx, subscriptions, timeout)
				if err != nil {
					return sqerrors.Wrap(err, "WAF execution error")
				}

				if blocked {
					spanErr = sdk_types.SqreenError{Err: ErrWAFProtection}
				}
				return nil
			})

			return spanErr
		}

		if err := runWAF(s); err != nil {
			return err
		}

		s.OnNewChild(func(s span.EmergingSpan) error {
			return runWAF(s)
		})

		s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
			return runWAF(data)
		})

		return nil
	})
}

func runReactiveWAF(attrs span.AttributeGetter, c CallbackContext, wafCtx waf_types.Rule, subscriptions []map[string]struct{}, timeout time.Duration) (blocked bool, err error) {
	p := c.ProtectionContext()

	// Check that we have `timeout` amount of time available
	if p.DeadlineExceeded(timeout) {
		return false, nil
	}

	args := make(waf_types.DataSet, len(subscriptions))
subscriptionLoop:
	for _, set := range subscriptions {
		for address := range set {
			if v, exists := attrs.Get(address); !exists || v == nil {
				// The full set is not present
				continue subscriptionLoop
			}
		}

		for address := range set {
			value, exists := attrs.Get(address)
			sqassert.True(exists)
			args[address] = value
			// TODO: have a separate set copy that where could remove the entries
			//  so that we avoid calling the WAF several times when the subscription
			//  has been already matched before.
		}

		// Check we haven't exceeded the time deadline if any
		if p.DeadlineExceeded(0) {
			return false, nil
		}
	}

	if len(args) == 0 {
		return false, nil
	}

	// TODO: if no timeout, remove the address from the subscriptions to avoid
	//   doing it again later
	return runWAF(c, wafCtx, args, timeout)
}

func runWAF(c CallbackContext, wafRule waf_types.Rule, args waf_types.DataSet, timeout time.Duration) (blocked bool, err error) {
	p := c.ProtectionContext()

	// Check that we have `timeout` amount of time available
	if p.DeadlineExceeded(timeout) {
		return false, nil
	}

	action, info, err := wafRule.Run(args, timeout)
	if err != nil {
		if err == waf_types.ErrTimeout {
			type errKey struct{}
			return false, sqerrors.WithKey(sqerrors.New("WAF timeout"), errKey{})
		}
		type errKey struct{}
		return false, sqerrors.WithKey(newWAFRunError(err, args, timeout), errKey{})
	}

	if action == waf_types.NoAction {
		return false, nil
	}

	attackInfo := api.WAFAttackInfo{WAFData: info}

	return c.HandleAttack(action == waf_types.BlockAction, event.WithAttackInfo(attackInfo)), nil
}

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// NewCallback returns the callback object or function for the given callback
// name. An error is returned if the callback name is unknown or an error
// occurred during the constructor call.
func NewNativeCallback(name string, ctx *nativeRuleContext, cfg callback.NativeCallbackConfig) (prolog sqhook.PrologCallback, err error) {
	var callbackCtor callback.NativeCallbackConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined native callback name `%s`", name)
	case "WriteCustomErrorPage", "WriteBlockingHTMLPage":
		ctx.SetCritical(true)
		callbackCtor = callback.NewWriteBlockingHTMLPageCallback
	case "WriteHTTPRedirection":
		ctx.SetCritical(true)
		callbackCtor = callback.NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbackCtor = callback.NewAddSecurityHeadersCallback
	case "MonitorHTTPStatusCode":
		ctx.SetCritical(true)
		callbackCtor = callback.NewMonitorHTTPStatusCodeCallback
	case "WAF":
		callbackCtor = callback.NewWAFCallback
	case "IPSecurityResponse":
		ctx.SetCritical(true)
		callbackCtor = callback.NewIPSecurityResponseCallback
	case "UserSecurityResponse":
		ctx.SetCritical(true)
		callbackCtor = callback.NewUserSecurityResponseCallback
	case "IPBlockList", "IPDenyList":
		ctx.SetCritical(true)
		callbackCtor = callback.NewIPDenyListCallback
	case "Shellshock":
		callbackCtor = callback.NewShellshockCallback
	}
	return callbackCtor(ctx, cfg)
}

// NewReflectedCallback returns the callback object or function of the given
// callback name. An error is returned if the callback name is unknown or an
// error occurred during the constructor call.
func NewReflectedCallback(name string, r callback.RuleContext, rule *api.Rule) (prolog sqhook.PrologCallback, err error) {
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined reflected callback name `%s`", name)
	case "", "JSExec":
		cfg, err := newJSReflectedCallbackConfig(rule)
		if err != nil {
			return nil, sqerrors.Wrap(err, "configuration error")
		}
		return callback.NewJSExecCallback(r, cfg)

	case "FunctionWAF":
		cfg, err := newReflectedCallbackConfig(rule)
		if err != nil {
			return nil, sqerrors.Wrap(err, "configuration error")
		}
		callbacks, ok := rule.Callbacks.RuleCallbacksNode.(*api.RuleFunctionWAFCallbacks)
		if !ok {
			return nil, sqerrors.Errorf("unexpected callbacks type `%T` instead of `%T`", rule.Callbacks.RuleCallbacksNode, callbacks)
		}
		return callback.NewFunctionWAFCallback(r, cfg, callbacks)
	}
}

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
func NewNativeCallback(name string, ctx callback.RuleFace, cfg callback.NativeCallbackConfig) (prolog sqhook.PrologCallback, err error) {
	var callbackCtor callback.NativeCallbackConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined native callback name `%s`", name)
	case "WriteCustomErrorPage":
		callbackCtor = callback.NewWriteCustomErrorPageCallback
	case "WriteHTTPRedirection":
		callbackCtor = callback.NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbackCtor = callback.NewAddSecurityHeadersCallback
	case "MonitorHTTPStatusCode":
		callbackCtor = callback.NewMonitorHTTPStatusCodeCallback
	case "WAF":
		callbackCtor = callback.NewWAFCallback
	case "IPSecurityResponse":
		callbackCtor = callback.NewIPSecurityResponseCallback
	case "UserSecurityResponse":
		callbackCtor = callback.NewUserSecurityResponseCallback
	case "IPBlockList", "IPDenyList":
		callbackCtor = callback.NewIPDenyListCallback
	case "Shellshock":
		callbackCtor = callback.NewShellshockCallback
	}
	return callbackCtor(ctx, cfg)
}

// NewReflectedCallback returns the callback object or function of the given
// callback name. An error is returned if the callback name is unknown or an
// error occurred during the constructor call.
func NewReflectedCallback(name string, ctx callback.RuleFace, r *api.Rule) (prolog sqhook.PrologCallback, err error) {
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined reflected callback name `%s`", name)
	case "", "JSExec":
		cfg, err := newJSReflectedCallbackConfig(r)
		if err != nil {
			return nil, sqerrors.Wrap(err, "configuration error")
		}
		return callback.NewJSExecCallback(ctx, cfg)

	case "FunctionWAF":
		cfg, err := newReflectedCallbackConfig(r)
		if err != nil {
			return nil, sqerrors.Wrap(err, "configuration error")
		}
		callbacks, ok := r.Callbacks.RuleCallbacksNode.(*api.RuleFunctionWAFCallbacks)
		if !ok {
			return nil, sqerrors.Errorf("unexpected callbacks type `%T` instead of `%T`", r.Callbacks.RuleCallbacksNode, callbacks)
		}
		return callback.NewFunctionWAFCallback(ctx, cfg, callbacks)
	}
}

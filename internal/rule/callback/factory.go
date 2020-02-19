// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"reflect"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// NewCallback returns the callback object or function for the given callback
// name. An error is returned if the callback name is unknown or an error
// occurred during the constructor call.
func NewNativeCallback(name string, ctx RuleFace) (prolog sqhook.PrologCallback, err error) {
	var callbackCtor NativeCallbackConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined native callback name `%s`", name)
	case "WriteCustomErrorPage":
		callbackCtor = NewWriteCustomErrorPageCallback
	case "WriteHTTPRedirection":
		callbackCtor = NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbackCtor = NewAddSecurityHeadersCallback
	case "MonitorHTTPStatusCode":
		callbackCtor = NewMonitorHTTPStatusCodeCallback
	case "WAF":
		callbackCtor = NewWAFCallback
	case "IPSecurityResponse":
		callbackCtor = NewIPSecurityResponseCallback
	case "UserSecurityResponse":
		callbackCtor = NewUserSecurityResponseCallback
	}
	return callbackCtor(ctx)
}

// NewReflectedCallback returns the callback object or function for the given callback
// name. An error is returned if the callback name is unknown or an error
// occurred during the constructor call.
func NewReflectedCallback(name string, prologFuncType reflect.Type, ctx RuleFace) (prolog sqhook.ReflectedPrologCallback, err error) {
	var callbackCtor ReflectedCallbackConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined reflected callback name `%s`", name)
	case "":
		fallthrough
	case "JSExec":
		callbackCtor = NewJSExecCallback
	}
	return callbackCtor(ctx, prologFuncType)
}

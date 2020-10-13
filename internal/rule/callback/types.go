// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/dop251/goja"
	"github.com/sqreen/go-agent/internal/backend/api"
	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// Config is the interface of the rule configuration.
type NativeCallbackConfig interface {
	BlockingMode() bool
	Data() interface{}
}

type ReflectedCallbackConfig interface {
	NativeCallbackConfig
	Strategy() *api.ReflectedCallbackConfig
}

type JSReflectedCallbackConfig interface {
	ReflectedCallbackConfig
	Pre() (funcDecl *goja.Program, funcCallParams []bindingaccessor.BindingAccessorFunc)
	Post() (funcDecl *goja.Program, funcCallParams []bindingaccessor.BindingAccessorFunc)
}

// NativeCallbackConstructorFunc is a function returning a native callback
// function or a CallbackObject.
type NativeCallbackConstructorFunc func(r RuleContext, cfg NativeCallbackConfig) (prolog sqhook.PrologCallback, err error)

// ReflectedCallbackConstructorFunc is a function returning a reflected callback
// function for the provided type.
type ReflectedCallbackConstructorFunc func(r RuleContext, cfg ReflectedCallbackConfig) (prolog sqhook.ReflectedPrologCallback, err error)

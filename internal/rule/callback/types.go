// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// RuleFace is the callback context of things it can use at run time.
type RuleFace interface {
	// Push a new metrics value for the given key into the default metrics store
	// given by the rule.
	// TODO: this should be instead performed when the request context is
	//   closed by checking the metrics stored in the event record and
	//   pushing them. But the current metrics store interface doesn't allow
	//   to pass a time (to pass the time of the observation).
	PushMetricsValue(key interface{}, value int64) error
	// TODO: variadic options api
	NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent
}

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
type NativeCallbackConstructorFunc func(r RuleFace, cfg NativeCallbackConfig) (prolog sqhook.PrologCallback, err error)

// ReflectedCallbackConstructorFunc is a function returning a reflected callback
// function for the provided type.
type ReflectedCallbackConstructorFunc func(r RuleFace, cfg ReflectedCallbackConfig) (prolog sqhook.ReflectedPrologCallback, err error)

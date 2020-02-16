// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"reflect"

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
	PushMetricsValue(key interface{}, value uint64) error
	Config() Config
	NewAttackEvent(blocked bool, info interface{}) *event.AttackEvent
}

// Config is the interface of the rule configuration.
type Config interface {
	BlockingMode() bool
	Data() interface{}
}

// NativeCallbackConstructorFunc is a function returning a native callback
// function or a CallbackObject.
type NativeCallbackConstructorFunc func(r RuleFace) (prolog sqhook.PrologCallback, err error)

// ReflectedCallbackConstructorFunc is a function returning a reflected callback
// function for the provided type.
type ReflectedCallbackConstructorFunc func(r RuleFace, prologFuncType reflect.Type) (prolog sqhook.ReflectedPrologCallback, err error)

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/internal/plog"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
)

// TODO: rename to RuleContext if we don't do ReflectedRuleContext for now
// TODO: add return values when generics allow to safely return the prolog type
type NativeRuleContext interface {
	Pre(func(c CallbackContext))
	Post(func(c CallbackContext))
}

type CallbackContext interface {
	ProtectionContext() protection_context.ProtectionContext

	// Push a new metrics value for the given key into the default metrics store
	// given by the rule.
	// TODO: this should be instead performed when the request context is
	//   closed by checking the metrics stored in the event record and
	//   pushing them. But the current metrics store interface doesn't allow
	//   to pass a time (to pass the time of the observation).
	PushMetricsValue(key interface{}, value int64) error
	Logger() Logger
	HandleAttack(shouldBock bool, info interface{}) (blocked bool)
}

type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

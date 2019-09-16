// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/record"
)

type Context interface {
	// Get the rule configuration.
	Config() interface{}
	// Push a new metrics value for the given key into the default metrics store
	// given by the rule.
	PushMetricsValue(key interface{}, value uint64)
	// NewAttack creates and returns a new attack event linked to the rule.
	NewAttack(blocked bool, infos interface{}) *record.AttackEvent
	// ErrorLogger allows to log errors from callbacks. It is restricted to
	// errors only as logs are expensive and should not be used for debugging
	// logs.
	plog.ErrorLogger
}

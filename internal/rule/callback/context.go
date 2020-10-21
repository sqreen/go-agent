// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/rule/callback/types"
)

type (
	RuleContext interface {
		Pre(pre CallbackFunc)
		Post(post CallbackFunc)
	}
	CallbackFunc = func(c CallbackContext)
)

type CallbackContext interface {
	ProtectionContext() types.ProtectionContext
	AddMetricsValue(key interface{}, value int64) error
	HandleAttack(shouldBock bool, info interface{}) (blocked bool)
	Logger() Logger
}

type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

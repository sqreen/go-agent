// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/plog"
	protection_types "github.com/sqreen/go-agent/internal/protection/types"
)

type (
	RuleContext interface {
		Pre(pre CallbackFunc)
		Post(post CallbackFunc)
	}
	CallbackFunc = func(c CallbackContext) error
)

type CallbackContext interface {
	HandleAttack(shouldBock bool, opt ...event.AttackEventOption) (blocked bool)
	ProtectionContext() ProtectionContext
	AddMetricsValue(key interface{}, value uint64) bool
	Logger() Logger
}

type ProtectionContext interface {
	protection_types.ProtectionContext
}

type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

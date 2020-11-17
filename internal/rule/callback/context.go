// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
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
	AddMetricsValue(key interface{}, value uint64) error
	Logger() Logger
}

type ProtectionContext interface {
	AddRequestParam(name string, v interface{})
	ClientIP() net.IP
	SqreenTime() *sqtime.SharedStopWatch
	DeadlineExceeded(needed time.Duration) (exceeded bool)
}

type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

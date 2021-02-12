// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import (
	"context"
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

type ProtectionContext interface {
	Context() context.Context
	ClientIP() net.IP
	SqreenTime() *sqtime.SharedStopWatch
	DeadlineExceeded(needed time.Duration) (exceeded bool)
	HandleAttack(block bool, attack *event.AttackEvent) (blocked bool)
	SetRequestBindingAccessorValue(address string, value interface{})
}

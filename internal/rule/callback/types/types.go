// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import (
	"net"
	"time"

	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

func FromGLS() ProtectionContext {
	v := sqgls.Get()
	actual, _ := v.(ProtectionContext)
	return actual
}

type ProtectionContext interface {
	AddRequestParam(name string, v interface{})
	HandleAttack(block bool, attack interface{}) (blocked bool)
	ClientIP() net.IP
	SqreenTime() *sqtime.SharedStopWatch
	DeadlineExceeded(needed time.Duration) (exceeded bool)
}

// Static assert that protection contexts correctly implement the
// ProtectionContext interface
var _ ProtectionContext = (*http_protection.ProtectionContext)(nil)

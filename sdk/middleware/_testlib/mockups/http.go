// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package mockups

import (
	"context"
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/plog"
	http_protection_types "github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
	"github.com/stretchr/testify/mock"
)

func NewRootHTTPProtectionContextMockup(ctx context.Context, ip string, path string) *RootHTTPProtectionContextMockup {
	r := &RootHTTPProtectionContextMockup{}

	r.ExpectIsPathAllowed(path).Return(false)
	r.ExpectIsIPAllowed(ip).Return(false)

	cfg := NewHTTPProtectionConfigMockup()
	r.ExpectConfig().Return(cfg)

	r.ExpectContext().Return(ctx)

	return r
}

type RootHTTPProtectionContextMockup struct {
	mock.Mock
}

func (a *RootHTTPProtectionContextMockup) FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error) {
	rets := a.Called(ip)
	if v := rets.Get(0); v != nil {
		action = v.(actor.Action)
	}
	exists = rets.Bool(1)
	err = rets.Error(2)
	return
}

func (a *RootHTTPProtectionContextMockup) FindActionByUserID(userID map[string]string) (action actor.Action, exists bool) {
	rets := a.Called(userID)
	if v := rets.Get(0); v != nil {
		action = v.(actor.Action)
	}
	exists = rets.Bool(1)
	return
}

func (a *RootHTTPProtectionContextMockup) Logger() *plog.Logger {
	if v := a.Called().Get(0); v != nil {
		return v.(*plog.Logger)
	}
	return nil
}

func (a *RootHTTPProtectionContextMockup) Config() http_protection_types.ConfigReader {
	if v := a.Called().Get(0); v != nil {
		return v.(http_protection_types.ConfigReader)
	}
	return nil
}

func (a *RootHTTPProtectionContextMockup) ExpectConfig() *mock.Call {
	return a.On("Config")
}

func (a *RootHTTPProtectionContextMockup) IsIPAllowed(ip net.IP) bool {
	return a.Called(ip).Bool(0)
}

func (a *RootHTTPProtectionContextMockup) ExpectIsIPAllowed(ip interface{}) *mock.Call {
	return a.On("IsIPAllowed", ip)
}

func (a *RootHTTPProtectionContextMockup) IsPathAllowed(path string) bool {
	return a.Called(path).Bool(0)
}

func (a *RootHTTPProtectionContextMockup) ExpectIsPathAllowed(path string) *mock.Call {
	return a.On("IsPathAllowed", path)
}

func (a *RootHTTPProtectionContextMockup) SqreenTime() *sqtime.SharedStopWatch {
	v, _ := a.Called().Get(0).(*sqtime.SharedStopWatch)
	return v
}

func (a *RootHTTPProtectionContextMockup) ExpectSqreenTime() *mock.Call {
	return a.On("SqreenTime")
}

func (a *RootHTTPProtectionContextMockup) DeadlineExceeded(needed time.Duration) (exceeded bool) {
	return a.Called(needed).Bool(0)
}

func (a *RootHTTPProtectionContextMockup) Context() context.Context {
	c, _ := a.Called().Get(0).(context.Context)
	return c
}

func (a *RootHTTPProtectionContextMockup) ExpectContext() *mock.Call {
	return a.On("Context")
}

func (a *RootHTTPProtectionContextMockup) CancelContext() {
	a.Called()
}

func (a *RootHTTPProtectionContextMockup) ExpectCancelContext() *mock.Call {
	return a.On("CancelContext")
}

func (a *RootHTTPProtectionContextMockup) Close(closed http_protection_types.ClosedProtectionContextFace) {
	a.Called(closed)
}

func (a *RootHTTPProtectionContextMockup) ExpectClose(v interface{}) {
	a.ExpectSqreenTime().Return(&sqtime.SharedStopWatch{})
	a.On("Close", v)
}

type HTTPProtectionConfigMockup struct {
	mock.Mock
}

func NewHTTPProtectionConfigMockup() *HTTPProtectionConfigMockup {
	m := &HTTPProtectionConfigMockup{}
	m.ExpectHTTPClientIPHeader().Return("").Maybe()
	m.ExpectHTTPClientIPHeaderFormat().Return("").Maybe()
	return m
}

func (c *HTTPProtectionConfigMockup) HTTPClientIPHeader() string {
	return c.Called().String(0)
}

func (c *HTTPProtectionConfigMockup) ExpectHTTPClientIPHeader() *mock.Call {
	return c.On("HTTPClientIPHeader")
}

func (c *HTTPProtectionConfigMockup) HTTPClientIPHeaderFormat() string {
	return c.Called().String(0)
}

func (c *HTTPProtectionConfigMockup) ExpectHTTPClientIPHeaderFormat() *mock.Call {
	return c.On("HTTPClientIPHeaderFormat")
}

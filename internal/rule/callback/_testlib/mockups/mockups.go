// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package mockups

import (
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
	"github.com/stretchr/testify/mock"
)

type NativeCallbackConfigMockup struct {
	mock.Mock
}

func (m *NativeCallbackConfigMockup) Data() interface{} {
	return m.Called().Get(0)
}

func (m *NativeCallbackConfigMockup) ExpectData() *mock.Call {
	return m.On("Data")
}

func (m *NativeCallbackConfigMockup) BlockingMode() bool {
	return m.Called().Bool(0)
}

func (m *NativeCallbackConfigMockup) ExpectBlockingMode() *mock.Call {
	return m.On("BlockingMode")
}

type NativeRuleContextMockup struct {
	mock.Mock
}

func (r *NativeRuleContextMockup) Pre(cb func(c callback.CallbackContext) error) {
	r.Called(cb)
}

func (r *NativeRuleContextMockup) ExpectPre(cb interface{}) *mock.Call {
	return r.On("Pre", cb)
}

func (r *NativeRuleContextMockup) Post(cb func(c callback.CallbackContext) error) {
	r.Called(cb)
}

func (r *NativeRuleContextMockup) ExpectPost(cb interface{}) *mock.Call {
	return r.On("Post", cb)
}

type CallbackContextMockup struct {
	mock.Mock
}

func (c *CallbackContextMockup) ProtectionContext() callback.ProtectionContext {
	v, _ := c.Called().Get(0).(callback.ProtectionContext)
	return v
}

func (c *CallbackContextMockup) ExpectProtectionContext() *mock.Call {
	return c.On("ProtectionContext")
}

func (c *CallbackContextMockup) Logger() callback.Logger {
	return c.Called().Get(0).(callback.Logger)
}

func (c *CallbackContextMockup) AddMetricsValue(key interface{}, value uint64) bool {
	return c.Called(key, value).Bool(0)
}

func (c *CallbackContextMockup) ExpectAddMetricsValue(key interface{}, value uint64) *mock.Call {
	return c.On("AddMetricsValue", key, value)
}

func (c *CallbackContextMockup) HandleAttack(shouldBock bool, opts ...event.AttackEventOption) (blocked bool) {
	return c.Called(shouldBock, opts).Bool(0)
}

func (c *CallbackContextMockup) ExpectHandleAttack(shouldBock bool, opts interface{}) *mock.Call {
	return c.On("HandleAttack", shouldBock, opts)
}

type ProtectionContextMockup struct {
	mock.Mock
}

var _ callback.ProtectionContext = (*ProtectionContextMockup)(nil)

func (p *ProtectionContextMockup) AddRequestParam(name string, v interface{}) {
	p.Called(name, v)
}

func (p *ProtectionContextMockup) ExpectAddRequestParam(name string, v interface{}) *mock.Call {
	return p.On("AddRequestParam", name, v)
}

func (p *ProtectionContextMockup) HandleAttack(block bool, attack *event.AttackEvent) (blocked bool) {
	return p.Called(block, attack).Bool(0)
}

func (p *ProtectionContextMockup) ExpectHandleAttack(block bool, opts ...event.AttackEventOption) *mock.Call {
	return p.On("HandleAttack", block, opts)
}

func (p *ProtectionContextMockup) ClientIP() net.IP {
	return p.Called().Get(0).(net.IP)
}

func (p *ProtectionContextMockup) ExpectClientIP() *mock.Call {
	return p.On("ClientIP")
}

func (p *ProtectionContextMockup) SqreenTime() *sqtime.SharedStopWatch {
	return p.Called().Get(0).(*sqtime.SharedStopWatch)
}

func (p *ProtectionContextMockup) ExpectSqreenTime() *mock.Call {
	return p.On("SqreenTime")
}

func (p *ProtectionContextMockup) DeadlineExceeded(needed time.Duration) (exceeded bool) {
	return p.Called(needed).Bool(0)
}

func (p *ProtectionContextMockup) ExpectDeadlineExceeded(needed time.Duration) *mock.Call {
	return p.On("DeadlineExceeded", needed)
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/event"
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

type RuleContextMockup struct {
	mock.Mock
}

func (r *RuleContextMockup) MonitorPre() {
	r.Called()
}

func (r *RuleContextMockup) PushMetricsValue(key interface{}, value int64) error {
	return r.Called(key, value).Error(0)
}

func (r *RuleContextMockup) ExpectPushMetricsValue(key interface{}, value int64) *mock.Call {
	return r.On("PushMetricsValue", key, value)
}

func (r *RuleContextMockup) NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent {
	return r.Called(blocked, info, st).Get(0).(*event.AttackEvent)
}

func (r *RuleContextMockup) ExpectNewAttackEvent(blocked bool, info interface{}) *mock.Call {
	return r.On("NewAttackEvent", blocked, info)
}

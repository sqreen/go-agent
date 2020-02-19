// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	CallbacksCtor              callback.NativeCallbackConstructorFunc
	ExpectEpilog, ExpectProlog bool
	InvalidTestCases           []interface{}
	ValidTestCases             []ValidTestCase
}

type ValidTestCase struct {
	Rule                       *RuleContextMockup
	TestCallback               func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback)
	ExpectAbortedCallbackChain bool
}

type callbackConfig struct {
	data interface{}
}

func (c callbackConfig) BlockingMode() bool {
	return false
}

func (c callbackConfig) Data() interface{} {
	return c.data
}

func (c callbackConfig) Strategy() *api.ReflectedCallbackConfig {
	return nil
}

func RunCallbackTest(t *testing.T, config TestConfig) {
	for _, data := range config.InvalidTestCases {
		data := data
		t.Run("with incorrect data", func(t *testing.T) {
			cbCfg := callbackConfig{data}
			prolog, err := config.CallbacksCtor(&RuleContextMockup{config: cbCfg})
			require.Error(t, err)
			require.Nil(t, prolog)
		})
	}

	for _, tc := range config.ValidTestCases {
		tc := tc
		t.Run("with correct data", func(t *testing.T) {
			// Instantiate the callback with the given correct rule data
			prolog, err := config.CallbacksCtor(tc.Rule)
			require.NoError(t, err)
			checkCallbacksValues(t, config, prolog)
			tc.TestCallback(t, tc.Rule, prolog)
		})
	}
}

func checkCallbacksValues(t *testing.T, config TestConfig, prolog sqhook.PrologCallback) {
	if config.ExpectProlog || config.ExpectEpilog {
		require.NotNil(t, prolog)
	}
}

type RuleContextMockup struct {
	config interface{}
	mock.Mock
}

func (m *RuleContextMockup) BlockingMode() bool {
	return m.Called().Bool(0)
}

func (m *RuleContextMockup) ExpectBlockingMode() *mock.Call {
	return m.On("BlockingMode")
}

func (mockup *RuleContextMockup) Error(err error) {
}

func (r *RuleContextMockup) PushMetricsValue(key interface{}, value uint64) error {
	return r.Called(key, value).Error(0)
}

func (r *RuleContextMockup) Config() callback.Config {
	return callbackConfig{data: r.config}
}

func (r *RuleContextMockup) NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent {
	return r.Called(blocked, info, st).Get(0).(*event.AttackEvent)
}

func (r *RuleContextMockup) ExpectNewAttackEvent(blocked bool, info interface{}) *mock.Call {
	return r.On("NewAttackEvent", blocked, info)
}

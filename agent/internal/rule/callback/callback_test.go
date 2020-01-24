// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/sqreen/go-agent/agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	CallbacksCtor              rule.CallbacksConstructorFunc
	ExpectEpilog, ExpectProlog bool
	PrologType, EpilogType     reflect.Type
	InvalidTestCases           []interface{}
	ValidTestCases             []ValidTestCase
}

type ValidTestCase struct {
	Rule                       *RuleContextMockup
	TestCallback               func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback)
	ExpectAbortedCallbackChain bool
}

func RunCallbackTest(t *testing.T, config TestConfig) {
	for _, data := range config.InvalidTestCases {
		data := data
		t.Run("with incorrect data", func(t *testing.T) {
			prolog, err := config.CallbacksCtor(&RuleContextMockup{config: data}, nil)
			require.Error(t, err)
			require.Nil(t, prolog)
		})
	}

	for _, tc := range config.ValidTestCases {
		tc := tc
		t.Run("with correct data", func(t *testing.T) {
			t.Run("without next callback", func(t *testing.T) {
				// Instantiate the callback with the given correct rule data
				prolog, err := config.CallbacksCtor(tc.Rule, nil)
				require.NoError(t, err)
				if actual, ok := prolog.(rule.CallbackObject); ok {
					prolog = actual.Prolog()
					defer actual.Close()
				}
				checkCallbacksValues(t, config, prolog)
				tc.TestCallback(t, tc.Rule, prolog)
			})

			t.Run("with next callback", func(t *testing.T) {
				t.Run("with wrong next prolog type", func(t *testing.T) {
					prolog, err := config.CallbacksCtor(tc.Rule, 33)
					require.Error(t, err)
					require.Nil(t, prolog)
				})

				t.Run("with correct next prolog", func(t *testing.T) {
					var called bool
					nextProlog := reflect.MakeFunc(config.PrologType, func(args []reflect.Value) (results []reflect.Value) {
						called = true
						return []reflect.Value{
							reflect.Zero(config.EpilogType),
							reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()),
						}
					}).Interface()

					prolog, err := config.CallbacksCtor(tc.Rule, nextProlog)
					require.NoError(t, err)
					if actual, ok := prolog.(rule.CallbackObject); ok {
						prolog = actual.Prolog()
						defer actual.Close()
					}

					checkCallbacksValues(t, config, prolog)
					require.NotNil(t, prolog)
					tc.TestCallback(t, tc.Rule, prolog)

					// Check the next prolog call when expected, ie. when the callback
					// did not abort.
					if tc.ExpectAbortedCallbackChain {
						require.False(t, called)
					} else {
						require.True(t, called)
					}
				})

				t.Run("with correct next epilog", func(t *testing.T) {
					var calledProlog, calledEpilog bool
					nextEpilog := reflect.MakeFunc(config.EpilogType, func(args []reflect.Value) (results []reflect.Value) {
						calledEpilog = true
						return
					})

					nextProlog := reflect.MakeFunc(config.PrologType, func(args []reflect.Value) (results []reflect.Value) {
						calledProlog = true
						return []reflect.Value{
							nextEpilog,
							reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()),
						}
					}).Interface()

					prolog, err := config.CallbacksCtor(tc.Rule, nextProlog)
					require.NoError(t, err)
					if actual, ok := prolog.(rule.CallbackObject); ok {
						prolog = actual.Prolog()
						defer actual.Close()
					}

					checkCallbacksValues(t, config, prolog)
					tc.TestCallback(t, tc.Rule, prolog)

					// Check the next prolog call when expected, ie. when the callback
					// did not abort.
					if tc.ExpectAbortedCallbackChain {
						require.False(t, calledProlog)
						require.False(t, calledEpilog)
					} else {
						require.True(t, calledProlog)
						require.True(t, calledEpilog)
					}
				})
			})
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

func (r *RuleContextMockup) PushMetricsValue(key interface{}, value uint64) {
	r.Called(key, value)
}

func (r *RuleContextMockup) Config() interface{} {
	return r.config
}

func (r *RuleContextMockup) NewAttack(blocked bool, info interface{}) *record.AttackEvent {
	return r.Called(blocked, info).Get(0).(*record.AttackEvent)
}

func (r *RuleContextMockup) ExpectNewAttack(blocked bool, info interface{}) *mock.Call {
	return r.On("NewAttack", blocked, info)
}

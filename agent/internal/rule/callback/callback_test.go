// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
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
	Rule          *FakeRule
	TestCallbacks func(t *testing.T, rule *FakeRule, prolog, epilog sqhook.Callback)
}

func RunCallbackTest(t *testing.T, config TestConfig) {
	for _, data := range config.InvalidTestCases {
		data := data
		t.Run("with incorrect data", func(t *testing.T) {
			prolog, epilog, err := config.CallbacksCtor(&FakeRule{config: data}, nil, nil)
			require.Error(t, err)
			require.Nil(t, prolog)
			require.Nil(t, epilog)
		})
	}

	for _, tc := range config.ValidTestCases {
		tc := tc
		t.Run("with correct data", func(t *testing.T) {
			t.Run("without next callbacks", func(t *testing.T) {
				// Instantiate the callback with the given correct rule data
				prolog, epilog, err := config.CallbacksCtor(tc.Rule, nil, nil)
				require.NoError(t, err)
				checkCallbacksValues(t, config, prolog, epilog)
				tc.TestCallbacks(t, tc.Rule, prolog, epilog)
			})

			t.Run("with next callbacks", func(t *testing.T) {
				t.Run("wrong next prolog type", func(t *testing.T) {
					prolog, epilog, err := config.CallbacksCtor(tc.Rule, 33, nil)
					require.Error(t, err)
					require.Nil(t, prolog)
					require.Nil(t, epilog)
				})

				t.Run("wrong next epilog type", func(t *testing.T) {
					prolog, epilog, err := config.CallbacksCtor(tc.Rule, nil, func() {})
					require.Error(t, err)
					require.Nil(t, prolog)
					require.Nil(t, epilog)
				})

				t.Run("with correct next prolog", func(t *testing.T) {
					var called bool
					nextProlog := reflect.MakeFunc(config.PrologType, func(args []reflect.Value) (results []reflect.Value) {
						called = true
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					}).Interface()

					prolog, epilog, err := config.CallbacksCtor(tc.Rule, nextProlog, nil)
					require.NoError(t, err)
					checkCallbacksValues(t, config, prolog, epilog)
					require.NotNil(t, prolog)
					tc.TestCallbacks(t, tc.Rule, prolog, epilog)
					require.True(t, called)
				})

				t.Run("with correct next epilog", func(t *testing.T) {
					var called bool
					nextEpilog := reflect.MakeFunc(config.EpilogType, func(args []reflect.Value) (results []reflect.Value) {
						called = true
						return
					}).Interface()

					prolog, epilog, err := config.CallbacksCtor(tc.Rule, nil, nextEpilog)
					require.NoError(t, err)
					checkCallbacksValues(t, config, prolog, epilog)
					require.NotNil(t, epilog)
					tc.TestCallbacks(t, tc.Rule, prolog, epilog)
					require.True(t, called)
				})

				t.Run("with both correct next callbacks", func(t *testing.T) {
					var calledProlog, calledEpilog bool
					nextProlog := reflect.MakeFunc(config.PrologType, func(args []reflect.Value) (results []reflect.Value) {
						calledProlog = true
						return []reflect.Value{reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
					}).Interface()
					nextEpilog := reflect.MakeFunc(config.EpilogType, func(args []reflect.Value) (results []reflect.Value) {
						calledEpilog = true
						return
					}).Interface()

					prolog, epilog, err := config.CallbacksCtor(tc.Rule, nextProlog, nextEpilog)
					require.NoError(t, err)
					require.NotNil(t, prolog)
					require.NotNil(t, epilog)
					tc.TestCallbacks(t, tc.Rule, prolog, epilog)
					require.True(t, calledProlog)
					require.True(t, calledEpilog)
				})
			})
		})
	}
}

func checkCallbacksValues(t *testing.T, config TestConfig, prolog, epilog sqhook.Callback) {
	if config.ExpectProlog {
		require.NotNil(t, prolog)
	}
	if config.ExpectEpilog {
		require.NotNil(t, epilog)
	}
}

type FakeRule struct {
	config interface{}
	mock.Mock
}

func (r *FakeRule) AddMetricsValue(key interface{}, value uint64) {
	r.Called(key, value)
}

func (r *FakeRule) Config() interface{} {
	return r.config
}

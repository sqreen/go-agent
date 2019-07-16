// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

func TestNewAddSecurityHeadersCallbacks(t *testing.T) {
	RunCallbackTest(t, TestConfig{
		CallbacksCtor: callback.NewAddSecurityHeadersCallbacks,
		ExpectProlog:  true,
		PrologType:    reflect.TypeOf(callback.AddSecurityHeadersPrologCallbackType(nil)),
		EpilogType:    reflect.TypeOf(callback.AddSecurityHeadersEpilogCallbackType(nil)),
		InvalidTestCases: [][]interface{}{
			nil,
			{},
			{33},
			{"yet another wrong type"},
			{[]string{}},
			{nil},
			{[]string{"one"}},
			{[]string{"one", "two", "three"}},
		},
		ValidTestCases: []ValidTestCase{
			{
				ValidData: []interface{}{
					[]string{"k", "v"},
					[]string{"one", "two"},
					[]string{"canonical-header", "the value"},
				},
				TestCallbacks: func(t *testing.T, prolog, epilog sqhook.Callback) {
					expectedHeaders := http.Header{
						"K":                []string{"v"},
						"One":              []string{"two"},
						"Canonical-Header": []string{"the value"},
					}
					actualProlog, ok := prolog.(callback.AddSecurityHeadersPrologCallbackType)
					require.True(t, ok)
					var rec http.ResponseWriter = httptest.NewRecorder()
					err := actualProlog(nil, &rec)
					// Check it behaves as expected
					require.NoError(t, err)
					require.Equal(t, expectedHeaders, rec.Header())

					// Test the epilog if any
					if epilog != nil {
						actualEpilog, ok := epilog.(callback.AddSecurityHeadersEpilogCallbackType)
						require.True(t, ok)
						actualEpilog(&sqhook.Context{})
					}
				},
			},
		},
	})
}

type TestConfig struct {
	CallbacksCtor              rule.CallbacksConstructorFunc
	ExpectEpilog, ExpectProlog bool
	PrologType, EpilogType     reflect.Type
	InvalidTestCases           [][]interface{}
	ValidTestCases             []ValidTestCase
}

type ValidTestCase struct {
	ValidData     []interface{}
	TestCallbacks func(t *testing.T, prolog, epilog sqhook.Callback)
}

func RunCallbackTest(t *testing.T, config TestConfig) {
	for _, data := range config.InvalidTestCases {
		data := data
		t.Run("with incorrect data", func(t *testing.T) {
			prolog, epilog, err := config.CallbacksCtor(data, nil, nil)
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
				prolog, epilog, err := config.CallbacksCtor(tc.ValidData, nil, nil)
				require.NoError(t, err)
				checkCallbacksValues(t, config, prolog, epilog)
				tc.TestCallbacks(t, prolog, epilog)
			})

			t.Run("with next callbacks", func(t *testing.T) {
				t.Run("wrong next prolog type", func(t *testing.T) {
					prolog, epilog, err := config.CallbacksCtor(tc.ValidData, 33, nil)
					require.Error(t, err)
					require.Nil(t, prolog)
					require.Nil(t, epilog)
				})

				t.Run("wrong next epilog type", func(t *testing.T) {
					prolog, epilog, err := config.CallbacksCtor(tc.ValidData, nil, func() {})
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

					prolog, epilog, err := config.CallbacksCtor(tc.ValidData, nextProlog, nil)
					require.NoError(t, err)
					checkCallbacksValues(t, config, prolog, epilog)
					require.NotNil(t, prolog)
					tc.TestCallbacks(t, prolog, epilog)
					require.True(t, called)
				})

				t.Run("with correct next epilog", func(t *testing.T) {
					var called bool
					nextEpilog := reflect.MakeFunc(config.EpilogType, func(args []reflect.Value) (results []reflect.Value) {
						called = true
						return
					}).Interface()

					prolog, epilog, err := config.CallbacksCtor(tc.ValidData, nil, nextEpilog)
					require.NoError(t, err)
					checkCallbacksValues(t, config, prolog, epilog)
					require.NotNil(t, epilog)
					tc.TestCallbacks(t, prolog, epilog)
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

					prolog, epilog, err := config.CallbacksCtor(tc.ValidData, nextProlog, nextEpilog)
					require.NoError(t, err)
					require.NotNil(t, prolog)
					require.NotNil(t, epilog)
					tc.TestCallbacks(t, prolog, epilog)
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
		require.NotNil(t, prolog)
	}
}

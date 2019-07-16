// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

func TestNewWriteCustomErrorPageCallbacks(t *testing.T) {
	RunCallbackTest(t, TestConfig{
		CallbacksCtor: callback.NewWriteCustomErrorPageCallbacks,
		ExpectProlog:  true,
		PrologType:    reflect.TypeOf(callback.WriteCustomErrorPagePrologCallbackType(nil)),
		EpilogType:    reflect.TypeOf(callback.WriteCustomErrorPageEpilogCallbackType(nil)),
		InvalidTestCases: [][]interface{}{
			{33},
			{"yet another wrong type"},
		},
		ValidTestCases: []ValidTestCase{
			{
				ValidData:     nil,
				TestCallbacks: testWriteCustomErrorPageCallbacks(500),
			},
			{
				ValidData:     []interface{}{},
				TestCallbacks: testWriteCustomErrorPageCallbacks(500),
			},
			{
				ValidData: []interface{}{
					&api.CustomErrorPageRuleDataEntry{StatusCode: 33},
				},
				TestCallbacks: testWriteCustomErrorPageCallbacks(33),
			},
		},
	})
}

func testWriteCustomErrorPageCallbacks(expectedStatusCode int) func(t *testing.T, prolog sqhook.Callback, epilog sqhook.Callback) {
	return func(t *testing.T, prolog, epilog sqhook.Callback) {
		actualProlog, ok := prolog.(callback.WriteCustomErrorPagePrologCallbackType)
		require.True(t, ok)
		var (
			statusCode int
			body       []byte
		)
		err := actualProlog(nil, nil, nil, nil, &statusCode, &body)
		// Check it behaves as expected
		require.NoError(t, err)
		require.Equal(t, expectedStatusCode, statusCode)
		require.NotNil(t, body)

		// Test the epilog if any
		if epilog != nil {
			actualEpilog, ok := epilog.(callback.AddSecurityHeadersEpilogCallbackType)
			require.True(t, ok)
			actualEpilog(&sqhook.Context{})
		}
	}
}

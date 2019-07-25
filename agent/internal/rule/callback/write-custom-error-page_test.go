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
		InvalidTestCases: []interface{}{
			33,
			"yet another wrong type",
		},
		ValidTestCases: []ValidTestCase{
			{
				Rule:          &FakeRule{},
				TestCallbacks: testWriteCustomErrorPageCallbacks(500),
			},
			{
				Rule: &FakeRule{
					config: &api.CustomErrorPageRuleDataEntry{StatusCode: 33},
				},
				TestCallbacks: testWriteCustomErrorPageCallbacks(33),
			},
		},
	})
}

func testWriteCustomErrorPageCallbacks(expectedStatusCode int) func(t *testing.T, rule *FakeRule, prolog sqhook.PrologCallback) {
	return func(t *testing.T, _ *FakeRule, prolog sqhook.PrologCallback) {
		actualProlog, ok := prolog.(callback.WriteCustomErrorPagePrologCallbackType)
		require.True(t, ok)
		var (
			statusCode int
			body       []byte
		)
		epilog, err := actualProlog(nil, nil, nil, &statusCode, &body)
		// Check it behaves as expected
		require.NoError(t, err)
		require.Equal(t, expectedStatusCode, statusCode)
		require.NotNil(t, body)

		// Test the epilog if any
		if epilog != nil {
			require.True(t, ok)
			epilog()
		}
	}
}

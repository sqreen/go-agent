// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/stretchr/testify/require"
)

func TestNewWriteHTTPRedirectionCallbacks(t *testing.T) {
	RunCallbackTest(t, TestConfig{
		CallbacksCtor: callback.NewWriteHTTPRedirectionCallbacks,
		ExpectProlog:  true,
		PrologType:    reflect.TypeOf(callback.WriteHTTPRedirectionPrologCallbackType(nil)),
		EpilogType:    reflect.TypeOf(callback.WriteHTTPRedirectionEpilogCallbackType(nil)),
		InvalidTestCases: []interface{}{
			nil,
			33,
			"yet another wrong type",
			&api.CustomErrorPageRuleDataEntry{},
			&api.RedirectionRuleDataEntry{},
			&api.RedirectionRuleDataEntry{"http//sqreen.com"},
		},
		ValidTestCases: []ValidTestCase{
			{
				Rule: &RuleContextMockup{
					config: &api.RedirectionRuleDataEntry{"http://sqreen.com"},
				},
				TestCallback: func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback) {
					// Call it and check the behaviour follows the rule's data
					actualProlog, ok := prolog.(callback.WriteHTTPRedirectionPrologCallbackType)
					require.True(t, ok)
					var (
						statusCode int
						headers    http.Header
					)
					epilog, err := actualProlog(nil, nil, &headers, &statusCode, nil)
					// Check it behaves as expected
					require.NoError(t, err)
					require.Equal(t, http.StatusSeeOther, statusCode)
					require.NotNil(t, headers)
					require.Equal(t, "http://sqreen.com", headers.Get("Location"))

					// Test the epilog if any
					if epilog != nil {
						require.True(t, ok)
						epilog()
					}
				},
			},
		},
	})
}

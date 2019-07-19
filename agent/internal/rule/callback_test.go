// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule_test

import (
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/rule"
	"github.com/stretchr/testify/require"
)

func TestNewCallbacks(t *testing.T) {
	for _, tc := range []struct {
		testName      string
		name          string
		rule          *rule.CallbackContext
		shouldSucceed bool
	}{
		{
			testName:      "not existing",
			name:          "iDontExist",
			rule:          nil,
			shouldSucceed: false,
		},
		{
			testName:      "empty string",
			name:          "",
			rule:          nil,
			shouldSucceed: false,
		},
		{
			testName: "WriteCustomErrorPage",
			name:     "WriteCustomErrorPage",
			rule: rule.NewCallbackContext(&api.Rule{
				Data: api.RuleData{
					Values: []api.RuleDataEntry{
						{&api.CustomErrorPageRuleDataEntry{}},
					},
				},
			}, nil, nil),
			shouldSucceed: true,
		},
	} {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			_, _, err := rule.NewCallbacks(tc.name, tc.rule, nil, nil)
			if tc.shouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

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
		data          []interface{}
		shouldSucceed bool
	}{
		{
			testName:      "not existing",
			name:          "iDontExist",
			data:          nil,
			shouldSucceed: false,
		},
		{
			testName:      "empty string",
			name:          "",
			data:          nil,
			shouldSucceed: false,
		},
		{
			testName: "WriteCustomErrorPage",
			name:     "WriteCustomErrorPage",
			data: []interface{}{
				&api.CustomErrorPageRuleDataEntry{},
			},
			shouldSucceed: true,
		},
	} {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			_, _, err := rule.NewCallbacks(tc.name, tc.data, nil, nil)
			if tc.shouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/stretchr/testify/require"
)

func TestNewWriteCustomErrorPageCallbacks(t *testing.T) {
	t.Run("with incorrect data", func(t *testing.T) {
		for _, data := range [][]interface{}{
			{33},
			{"yet another wrong type"},
		} {
			prolog, epilog, err := callback.NewWriteCustomErrorPageCallbacks(data)
			require.Error(t, err)
			require.Nil(t, prolog)
			require.Nil(t, epilog)
		}
	})

	t.Run("with correct data", func(t *testing.T) {
		for _, tc := range []struct {
			testName           string
			data               []interface{}
			expectedStatusCode int
		}{
			{
				testName:           "default behaviour with nil data",
				data:               nil,
				expectedStatusCode: 500,
			},
			{
				testName:           "default behaviour with empty array",
				data:               nil,
				expectedStatusCode: 500,
			},
			{
				testName: "actual rule data",
				data: []interface{}{
					&api.CustomErrorPageRuleDataEntry{
						StatusCode: 33,
					},
				},
				expectedStatusCode: 33,
			},
		} {
			tc := tc
			t.Run(tc.testName, func(t *testing.T) {
				// Instantiate the callback with the given correct rule data
				prolog, epilog, err := callback.NewWriteCustomErrorPageCallbacks(tc.data)
				require.NoError(t, err)
				require.NotNil(t, prolog)
				require.Nil(t, epilog)
				// Call it and check the behaviour follows the rule's data
				actualProlog, ok := prolog.(callback.WriteCustomErrorPagePrologCallbackType)
				require.True(t, ok)
				var (
					statusCode int
					body       []byte
				)
				err = actualProlog(nil, nil, nil, nil, &statusCode, &body)
				// Check it behaves as expected
				require.NoError(t, err)
				require.Equal(t, tc.expectedStatusCode, statusCode)
				require.NotNil(t, body)
			})
		}
	})
}

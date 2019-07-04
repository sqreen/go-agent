// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/stretchr/testify/require"
)

func TestNewWriteHTTPRedirectionCallbacks(t *testing.T) {
	t.Run("with incorrect data", func(t *testing.T) {
		for _, data := range [][]interface{}{
			nil,
			{},
			{33},
			{"yet another wrong type"},
			{&api.CustomErrorPageRuleDataEntry{}},
			{&api.RedirectionRuleDataEntry{}},
			{&api.RedirectionRuleDataEntry{"http//sqreen.com"}},
		} {
			prolog, epilog, err := callback.NewWriteHTTPRedirectionCallbacks(data, nil, nil)
			require.Error(t, err)
			require.Nil(t, prolog)
			require.Nil(t, epilog)
		}
	})

	t.Run("with correct data", func(t *testing.T) {
		// Instantiate the callback with the given correct rule data
		expectedURL := "http://sqreen.com"
		prolog, epilog, err := callback.NewWriteHTTPRedirectionCallbacks([]interface{}{
			&api.RedirectionRuleDataEntry{RedirectionURL: expectedURL},
		}, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, prolog)
		require.Nil(t, epilog)
		// Call it and check the behaviour follows the rule's data
		actualProlog, ok := prolog.(callback.WriteHTTPRedirectionPrologCallbackType)
		require.True(t, ok)
		var (
			statusCode int
			headers    http.Header
		)
		err = actualProlog(nil, nil, nil, &headers, &statusCode, nil)
		// Check it behaves as expected
		require.NoError(t, err)
		require.Equal(t, http.StatusSeeOther, statusCode)
		require.NotNil(t, headers)
		require.Equal(t, expectedURL, headers.Get("Location"))
	})
}

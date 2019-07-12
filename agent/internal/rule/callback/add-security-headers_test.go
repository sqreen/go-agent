// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/stretchr/testify/require"
)

func TestNewAddSecurityHeadersCallbacks(t *testing.T) {
	t.Run("with incorrect data", func(t *testing.T) {
		for _, data := range [][]interface{}{
			nil,
			{},
			{33},
			{"yet another wrong type"},
			{[]string{}},
			{nil},
			{[]string{"one"}},
			{[]string{"one", "two", "three"}},
		} {
			prolog, epilog, err := callback.NewAddSecurityHeadersCallbacks(data)
			require.Error(t, err)
			require.Nil(t, prolog)
			require.Nil(t, epilog)
		}
	})

	t.Run("with correct data", func(t *testing.T) {
		// Instantiate the callback with the given correct rule data
		prolog, epilog, err := callback.NewAddSecurityHeadersCallbacks([]interface{}{
			[]string{"k", "v"},
			[]string{"one", "two"},
			[]string{"canonical-header", "the value"},
		})
		require.NoError(t, err)
		require.NotNil(t, prolog)
		require.Nil(t, epilog)
		// Call it and check the behaviour follows the rule's data
		actualProlog, ok := prolog.(callback.AddSecurityHeadersPrologCallbackType)
		require.True(t, ok)
		var rec http.ResponseWriter = httptest.NewRecorder()
		err = actualProlog(nil, &rec)
		// Check it behaves as expected
		require.NoError(t, err)
		expectedHeaders := http.Header{
			"K":                []string{"v"},
			"One":              []string{"two"},
			"Canonical-Header": []string{"the value"},
		}
		require.Equal(t, expectedHeaders, rec.Header())
	})
}

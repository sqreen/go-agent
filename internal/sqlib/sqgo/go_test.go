// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgo_test

import (
	"testing"

	"github.com/sqreen/go-agent/internal/sqlib/sqgo"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	t.Run("Unvendor", func(t *testing.T) {
		for _, tc := range []struct {
			Symbol   string
			Expected string
		}{
			{
				Symbol:   "github.com/sqreen/go-agent/internal/protection/http",
				Expected: "github.com/sqreen/go-agent/internal/protection/http",
			},

			{
				Symbol:   "github.com/my-org/my-app/vendor/github.com/sqreen/go-agent/internal/protection/http",
				Expected: "github.com/sqreen/go-agent/internal/protection/http",
			},

			{
				Symbol:   "my-app/vendor/github.com/sqreen/go-agent/internal/protection/http",
				Expected: "github.com/sqreen/go-agent/internal/protection/http",
			},

			{
				Symbol:   "my-app/vendor/github.com/sqreen/go-agent/internal/protection/http.(*ProtectionContext).Foo",
				Expected: "github.com/sqreen/go-agent/internal/protection/http.(*ProtectionContext).Foo",
			},

			{
				Symbol:   "github.com/sqreen/go-agent/internal/protection/http.(*ProtectionContext).Foo",
				Expected: "github.com/sqreen/go-agent/internal/protection/http.(*ProtectionContext).Foo",
			},
		} {
			tc := tc
			t.Run("", func(t *testing.T) {
				got := sqgo.Unvendor(tc.Symbol)
				require.Equal(t, tc.Expected, got)
			})
		}
	})
}

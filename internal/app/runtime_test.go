// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVendorPrefix(t *testing.T) {
	for _, tc := range []struct {
		PkgPath  string
		Expected string
	}{
		{
			PkgPath:  "github.com/sqreen/go-agent/internal/protection/http",
			Expected: "",
		},

		{
			PkgPath:  "github.com/my-org/my-app/vendor/github.com/sqreen/go-agent/internal/protection/http",
			Expected: "github.com/my-org/my-app/vendor/",
		},

		{
			PkgPath:  "my-app/vendor/github.com/sqreen/go-agent/internal/protection/http",
			Expected: "my-app/vendor/",
		},
	} {
		tc := tc
		t.Run("", func(t *testing.T) {
			got := vendorPrefix(tc.PkgPath)
			require.Equal(t, tc.Expected, got)
		})
	}
}

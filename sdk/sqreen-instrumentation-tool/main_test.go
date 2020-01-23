// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCommandID(t *testing.T) {
	var compilePath string
	if runtime.GOOS == "windows" {
		compilePath = "c:\\go\\pkg\\tool\\windows_amd64\\compile.exe"
	} else {
		compilePath = "/usr/lib/go-1.13/pkg/tool/linux_amd64/compile"
	}

	tests := []struct {
		arg     string
		want    string
		wantErr bool
	}{
		{
			arg:  compilePath,
			want: "compile",
		},
		{
			arg:     "",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.arg, func(t *testing.T) {
			got, err := parseCommandID(tc.arg)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"
	"testing"

	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	randomPackageStr := testlib.RandUTF8String()
	randomOutputStr := testlib.RandUTF8String()
	randomStr := testlib.RandUTF8String()

	tests := []struct {
		name                    string
		args                    []string
		expectedFlags           compileFlagSet
		expectedValidationError bool
	}{
		{
			name:                    "empty args",
			args:                    []string{},
			expectedFlags:           compileFlagSet{},
			expectedValidationError: true,
		},
		{
			name: "package option",
			args: []string{fmt.Sprintf("-p=%s", randomPackageStr)},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
			},
			expectedValidationError: true,
		},
		{
			name: "package option",
			args: []string{"-p", randomPackageStr},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
			},
			expectedValidationError: true,
		},
		{
			name: "output option",
			args: []string{fmt.Sprintf("-o=%s", randomOutputStr)},
			expectedFlags: compileFlagSet{
				Output: randomOutputStr,
			},
			expectedValidationError: true,
		},
		{
			name: "output option",
			args: []string{"-o", randomOutputStr},
			expectedFlags: compileFlagSet{
				Output: randomOutputStr,
			},
			expectedValidationError: true,
		},
		{
			name: "output and package options",
			args: []string{"-p", randomPackageStr, "-o", randomOutputStr},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
				Output:  randomOutputStr,
			},
		},
		{
			name: "output and package options",
			args: []string{fmt.Sprintf("-p=%s", randomPackageStr), fmt.Sprintf("-o=%s", randomOutputStr)},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
				Output:  randomOutputStr,
			},
		},
		{
			name: "output and package options",
			args: []string{"-o", randomOutputStr, fmt.Sprintf("-p=%s", randomPackageStr)},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
				Output:  randomOutputStr,
			},
		},
		{
			name: "output and package options",
			args: []string{fmt.Sprintf("-o=%s", randomOutputStr), "-p", randomPackageStr},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
				Output:  randomOutputStr,
			},
		},
		{
			name: "output and package options and others",
			args: []string{fmt.Sprintf("-o=%s", randomOutputStr), "-p", randomPackageStr, "-a", "-b", fmt.Sprintf("-c=%s", randomStr), "-d", randomStr, "a.go", "b.go", "c.go"},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
				Output:  randomOutputStr,
			},
		},
		{
			name: "empty output option value",
			args: []string{"-p", randomPackageStr, "-o", ""},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
			},
			expectedValidationError: true,
		},
		{
			name: "empty package option value",
			args: []string{"-o", randomOutputStr, "-p", ""},
			expectedFlags: compileFlagSet{
				Output: randomOutputStr,
			},
			expectedValidationError: true,
		},
		{
			name: "empty output option value",
			args: []string{"-p", randomPackageStr, "-o="},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
			},
			expectedValidationError: true,
		},
		{
			name: "empty package option value",
			args: []string{"-o", randomOutputStr, "-p="},
			expectedFlags: compileFlagSet{
				Output: randomOutputStr,
			},
			expectedValidationError: true,
		},
		{
			name: "missing package option value",
			args: []string{"-p", "-o", randomOutputStr},
			expectedFlags: compileFlagSet{
				Output: randomOutputStr,
			},
			expectedValidationError: true,
		},
		{
			name: "missing output option value",
			args: []string{"-p", randomPackageStr, "-o"},
			expectedFlags: compileFlagSet{
				Package: randomPackageStr,
			},
			expectedValidationError: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var flags compileFlagSet
			parseFlags(&flags, tt.args)
			require.Equal(t, tt.expectedFlags, flags)

			isValid := flags.IsValid()
			if tt.expectedValidationError {
				require.False(t, isValid)
			} else {
				require.True(t, isValid)
			}
		})
	}
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsanitize_test

import (
	"fmt"
	"net/url"
	"regexp"
	"testing"

	"github.com/sqreen/go-agent/agent/sqlib/sqsanitize"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestGoAssumptions(t *testing.T) {
	t.Run("an empty regular expression matches everything", func(t *testing.T) {
		re := regexp.MustCompile("")
		require.True(t, re.MatchString("hello"))
	})
}

func TestScrubber(t *testing.T) {
	expectedMask := `<Redacted by Sqreen>`

	randString := testlib.RandUTF8String(512, 1024)

	t.Run("NewScrubber", func(t *testing.T) {
		type args struct {
			keyRegexp         string
			valueRegexp       string
			redactedValueMask string
		}
		tests := []struct {
			name    string
			args    args
			want    *sqsanitize.Scrubber
			wantErr bool
		}{
			{
				name: "key regexp should not compile",
				args: args{
					keyRegexp:         "o(ops",
					valueRegexp:       "",
					redactedValueMask: expectedMask,
				},
				wantErr: true,
			},
			{
				name: "value regexp should not compile",
				args: args{
					keyRegexp:         "",
					valueRegexp:       "o(ops",
					redactedValueMask: expectedMask,
				},
				wantErr: true,
			},
			{
				name: "no regexps",
				args: args{
					keyRegexp:         "",
					valueRegexp:       "",
					redactedValueMask: expectedMask,
				},
			},
			{
				name: "key regexp only",
				args: args{
					keyRegexp:         "ok",
					valueRegexp:       "",
					redactedValueMask: expectedMask,
				},
			},
			{
				name: "value regexp only",
				args: args{
					keyRegexp:         "",
					valueRegexp:       "ok",
					redactedValueMask: expectedMask,
				},
			},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				got, err := sqsanitize.NewScrubber(tc.args.keyRegexp, tc.args.valueRegexp, tc.args.redactedValueMask)
				if tc.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					require.NotNil(t, got)
				}
			})
		}
	})

	t.Run("string regexps", func(t *testing.T) {
		tests := []struct {
			name        string
			valueRegexp string
			value       string
			expected    string
		}{
			{
				name:        "full string",
				valueRegexp: "^every+thing$",
				value:       "everyyyyyyyyyyyything",
				expected:    expectedMask,
			},
			{
				name:        "ends with",
				valueRegexp: "end$",
				value:       fmt.Sprintf("%send", randString),
				expected:    fmt.Sprintf("%s%s", randString, expectedMask),
			},
			{
				name:        "starts with",
				valueRegexp: "^start",
				value:       fmt.Sprintf("start%s", randString),
				expected:    fmt.Sprintf("%s%s", expectedMask, randString),
			},
			{
				name:        "every submatch",
				valueRegexp: "any*where",
				value:       fmt.Sprintf("%sanywhere%sanyyyyyywhere%sanwhere%s", randString, randString, randString, randString),
				expected:    fmt.Sprintf("%s%s%s%s%s%s%s", randString, expectedMask, randString, expectedMask, randString, expectedMask, randString),
			},
			{
				name:        "no match",
				valueRegexp: "anywhere",
				value:       randString,
				expected:    randString,
			},
			{
				name:        "disabled",
				valueRegexp: "",
				value:       randString,
				expected:    randString,
			},
		}
		for _, tc := range tests {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				s, err := sqsanitize.NewScrubber(testlib.RandUTF8String(), tc.valueRegexp, expectedMask)
				require.NoError(t, err)
				s.Scrub(&tc.value)
				require.Equal(t, tc.expected, tc.value)
			})

		}
	})

	t.Run("Scrub", func(t *testing.T) {
		type myStruct struct {
			a string
			B string
			C int
			D string
			E *myStruct
		}

		type TestCase struct {
			name     string
			value    interface{}
			expected interface{}
		}

		valueRegexp := `^everything$`

		tests := []TestCase{
			{
				name:     "string",
				value:    "",
				expected: "",
			},
			{
				name:     "string",
				value:    "not everything",
				expected: "not everything",
			},
			{
				name:     "string",
				value:    "everything",
				expected: expectedMask,
			},
			{
				name:     "slice",
				value:    nil,
				expected: nil,
			},
			{
				name:     "slice",
				value:    []string{},
				expected: []string{},
			},
			{
				name:     "slice",
				value:    []string{"f", "fo", "foo"},
				expected: []string{"f", "fo", "foo"},
			},
			{
				name:     "slice",
				value:    []string{"everything", "everithing", "not everything", "everything not", "everything"},
				expected: []string{expectedMask, "everithing", "not everything", "everything not", expectedMask},
			},
			{
				name:     "map",
				value:    nil,
				expected: nil,
			},
			{
				name:     "map",
				value:    map[string]string{},
				expected: map[string]string{},
			},
			{
				name:     "map",
				value:    map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
				expected: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
			},
			{
				name:     "map",
				value:    map[string]string{"key": "everything"},
				expected: map[string]string{"key": expectedMask},
			},
			{
				name:     "map",
				value:    map[string]string{"k1": "everything", "k2": "everithing", "k3": "not everything", "k4": "everything not", "k5": "everything"},
				expected: map[string]string{"k1": expectedMask, "k2": "everithing", "k3": "not everything", "k4": "everything not", "k5": expectedMask},
			},
			{
				name: "struct",
				value: &myStruct{
					a: "everything",
					B: "everything",
					C: 33,
					D: "not everything",
				},
				expected: &myStruct{
					a: "everything",
					B: expectedMask,
					C: 33,
					D: "not everything",
				},
			},
			{
				name: "struct",
				value: &myStruct{
					E: &myStruct{
						a: "everything",
						B: "everything",
						C: 33,
						D: "not everything",
					},
				},
				expected: &myStruct{
					E: &myStruct{
						a: "everything",
						B: expectedMask,
						C: 33,
						D: "not everything",
					},
				},
			},
		}

		s, err := sqsanitize.NewScrubber(testlib.RandUTF8String(), valueRegexp, expectedMask)
		require.NoError(t, err)
		for _, tc := range tests {
			tc := tc
			var value, expected interface{}
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				switch v := tc.value.(type) {
				case string:
					// Need a string that can be set - hence using the address of the
					// local variable v
					value = &v
					// expected must have the *string type in order to have deep equal
					// working
					want := tc.expected.(string)
					expected = &want
				default:
					value = tc.value
					expected = tc.expected
				}
				err := s.Scrub(value)
				require.NoError(t, err)
				require.Equal(t, expected, value)
			})

		}
	})

	t.Run("Usage", func(t *testing.T) {
		s, err := sqsanitize.NewScrubber("", "forbidden", expectedMask)
		require.NoError(t, err)

		t.Run("URL Values", func(t *testing.T) {
			values := url.Values{
				"password": []string{"no", "pass", randString, "forbidden", randString}, // TODO
				"other":    []string{"forbidden", "whatforbidden", randString, "key"},
				"":         []string{"forbidden", "forbiddenwhat", randString, "key"},
			}
			err := s.Scrub(values)
			require.NoError(t, err)
			expected := url.Values{
				"password": []string{"no", "pass", randString, expectedMask, randString}, // TODO
				"other":    []string{expectedMask, "what" + expectedMask, randString, "key"},
				"":         []string{expectedMask, expectedMask + "what", randString, "key"},
			}
			require.Equal(t, expected, values)
		})
	})
}

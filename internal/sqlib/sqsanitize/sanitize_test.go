// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsanitize_test

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/internal/sqlib/sqsanitize"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

func TestGoAssumptions(t *testing.T) {
	t.Run("an empty regular expression matches everything", func(t *testing.T) {
		re := regexp.MustCompile("")
		require.True(t, re.MatchString("hello"))
	})

	t.Run("a map entry cannot be set", func(t *testing.T) {
		v := map[string]*struct{}{
			"k": {},
		}
		require.False(t, reflect.ValueOf(&v).Elem().MapIndex(reflect.ValueOf("k")).CanSet())
	})

	t.Run("a slice entry can be set", func(t *testing.T) {
		v := []struct{}{{}}
		require.True(t, reflect.ValueOf(v).Index(0).CanSet())
	})

	t.Run("an array entry can be set", func(t *testing.T) {
		v := [1]*struct{}{{}}
		require.True(t, reflect.ValueOf(&v).Elem().Index(0).CanSet())
	})

	t.Run("an unexported struct field cannot be set", func(t *testing.T) {
		v := struct{ f *struct{} }{}
		require.False(t, reflect.ValueOf(&v).Elem().Field(0).CanSet())
	})

	t.Run("an exported struct field can be set", func(t *testing.T) {
		v := struct{ F *struct{} }{}
		require.True(t, reflect.ValueOf(&v).Elem().Field(0).CanSet())
	})
}

func TestScrubber(t *testing.T) {
	expectedMask := `<Redacted by Sqreen>`

	randString := testlib.RandUTF8String(512, 1024)
	//randString := "foo"

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
				info := sqsanitize.Info{}
				scrubbed, err := s.Scrub(&tc.value, info)
				require.NoError(t, err)
				require.Equal(t, tc.expected, tc.value)
				require.True(t, (scrubbed && len(info) > 0) || (!scrubbed && len(info) == 0))
			})

		}
	})

	t.Run("Scrub", func(t *testing.T) {
		// The following test checks the scrubbed value when using the key regexp,
		// or the value regexp or both.
		type expectedValues struct {
			withValueRE      interface{}
			withKeyRE        interface{}
			withBothRE       interface{}
			withBothDisabled interface{}
		}
		type testCase struct {
			name     string
			value    interface{}
			expected expectedValues
		}

		// Given the following regular expressions
		keyRegexp := `(?i)(passw(or)?d)|(secret)|(authorization)|(api_?key)|(access_?token)`
		valueRegexp := `^everything$`

		tests := []testCase{
			{
				name:  "string",
				value: "",
				expected: expectedValues{
					withValueRE:      "",
					withKeyRE:        "",
					withBothRE:       "",
					withBothDisabled: "",
				},
			},
			{
				name:  "string",
				value: randString,
				expected: expectedValues{
					withValueRE:      randString,
					withKeyRE:        randString,
					withBothRE:       randString,
					withBothDisabled: randString,
				},
			},
			{
				name:  "string",
				value: "everything",
				expected: expectedValues{
					withValueRE:      expectedMask,
					withKeyRE:        "everything",
					withBothRE:       expectedMask,
					withBothDisabled: "everything",
				},
			},
			{
				name:  "slice",
				value: nil,
				expected: expectedValues{
					withValueRE:      nil,
					withKeyRE:        nil,
					withBothRE:       nil,
					withBothDisabled: nil,
				},
			},
			{
				name:  "slice",
				value: []string{},
				expected: expectedValues{
					withValueRE:      []string{},
					withKeyRE:        []string{},
					withBothRE:       []string{},
					withBothDisabled: []string{},
				},
			},
			{
				name:  "slice",
				value: []string{"f", "fo", "foo"},
				expected: expectedValues{
					withValueRE:      []string{"f", "fo", "foo"},
					withKeyRE:        []string{"f", "fo", "foo"},
					withBothRE:       []string{"f", "fo", "foo"},
					withBothDisabled: []string{"f", "fo", "foo"},
				},
			},
			{
				name:  "slice",
				value: func() interface{} { return []string{"everything", "everithing", "not everything", "everything not", "everything"} },
				expected: expectedValues{
					withValueRE:      []string{expectedMask, "everithing", "not everything", "everything not", expectedMask},
					withKeyRE:        []string{"everything", "everithing", "not everything", "everything not", "everything"},
					withBothRE:       []string{expectedMask, "everithing", "not everything", "everything not", expectedMask},
					withBothDisabled: []string{"everything", "everithing", "not everything", "everything not", "everything"},
				},
			},
			{
				name:  "map",
				value: nil,
				expected: expectedValues{
					withValueRE:      nil,
					withKeyRE:        nil,
					withBothRE:       nil,
					withBothDisabled: nil,
				},
			},
			{
				name:  "map",
				value: map[string]string{},
				expected: expectedValues{
					withValueRE:      map[string]string{},
					withKeyRE:        map[string]string{},
					withBothRE:       map[string]string{},
					withBothDisabled: map[string]string{},
				},
			},
			{
				name:  "map",
				value: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
				expected: expectedValues{
					withValueRE:      map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
					withKeyRE:        map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
					withBothRE:       map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
					withBothDisabled: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
				},
			},
			{
				name:  "map",
				value: func() interface{} { return map[string]string{"key": "everything"} },
				expected: expectedValues{
					withValueRE:      map[string]string{"key": expectedMask},
					withKeyRE:        map[string]string{"key": "everything"},
					withBothRE:       map[string]string{"key": expectedMask},
					withBothDisabled: map[string]string{"key": "everything"},
				},
			},
			{
				name: "map",
				value: func() interface{} {
					return map[string]string{
						"key":    "everything",
						"passwd": randString,
						"apikey": randString,
						"k4":     "not everything",
						"secret": "everything",
					}
				},
				expected: expectedValues{
					withValueRE: map[string]string{
						"key":    expectedMask,
						"passwd": randString,
						"apikey": randString,
						"k4":     "not everything",
						"secret": expectedMask,
					},
					withKeyRE: map[string]string{
						"key":    "everything",
						"passwd": expectedMask,
						"apikey": expectedMask,
						"k4":     "not everything",
						"secret": expectedMask,
					},
					withBothRE: map[string]string{
						"key":    expectedMask,
						"passwd": expectedMask,
						"apikey": expectedMask,
						"k4":     "not everything",
						"secret": expectedMask,
					},
					withBothDisabled: map[string]string{
						"key":    "everything",
						"passwd": randString,
						"apikey": randString,
						"k4":     "not everything",
						"secret": "everything",
					},
				},
			},
			{
				name: "map of string pointers",
				value: func() interface{} {
					// New local variable to avoid modifying &randString
					// This test will modify &myRandString (a *string value)
					myRandString := randString
					return map[string]*string{
						"passwd": &myRandString,
						"apikey": &myRandString,
					}
				},
				expected: expectedValues{
					withValueRE: map[string]*string{
						"passwd": &randString,
						"apikey": &randString,
					},
					withKeyRE: map[string]*string{
						"passwd": &expectedMask,
						"apikey": &expectedMask,
					},
					withBothRE: map[string]*string{
						"passwd": &expectedMask,
						"apikey": &expectedMask,
					},
					withBothDisabled: map[string]*string{
						"passwd": &randString,
						"apikey": &randString,
					},
				},
			},
			{
				name: "struct",
				value: func() interface{} {
					return &myStruct{
						PassWORD: randString,
						password: randString,
						Secret_:  []string{randString, "everything"},
						ApiKey:   9838923,
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						PassWORD: randString,
						password: randString,
						Secret_:  []string{randString, expectedMask},
						ApiKey:   9838923,
					},
					withKeyRE: &myStruct{
						PassWORD: expectedMask,
						password: randString,
						Secret_:  []string{expectedMask, expectedMask},
						ApiKey:   9838923,
					},
					withBothRE: &myStruct{
						PassWORD: expectedMask,
						password: randString,
						Secret_:  []string{expectedMask, expectedMask},
						ApiKey:   9838923,
					},
					withBothDisabled: &myStruct{
						PassWORD: randString,
						password: randString,
						Secret_:  []string{randString, "everything"},
						ApiKey:   9838923,
					},
				},
			},
			{
				name: "struct",
				value: func() interface{} {
					return &myStruct{
						Face: myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, "everything"},
							ApiKey:   9838923,
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						Face: myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, expectedMask},
							ApiKey:   9838923,
						},
					},
					withKeyRE: &myStruct{
						Face: myStruct{PassWORD: expectedMask,
							password: randString,
							Secret_:  []string{expectedMask, expectedMask},
							ApiKey:   9838923,
						},
					},
					withBothRE: &myStruct{
						Face: myStruct{
							PassWORD: expectedMask,
							password: randString,
							Secret_:  []string{expectedMask, expectedMask},
							ApiKey:   9838923,
						},
					},
					withBothDisabled: &myStruct{
						Face: myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, "everything"},
							ApiKey:   9838923,
						},
					},
				},
			},
			{
				name: "struct",
				value: func() interface{} {
					return &myStruct{
						a: "everything",
						B: "everything",
						C: 33,
						D: "not everything",
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						a: "everything",
						B: expectedMask,
						C: 33,
						D: "not everything",
					},
					withKeyRE: &myStruct{
						a: "everything",
						B: "everything",
						C: 33,
						D: "not everything",
					},
					withBothRE: &myStruct{
						a: "everything",
						B: expectedMask,
						C: 33,
						D: "not everything",
					},
					withBothDisabled: &myStruct{
						a: "everything",
						B: "everything",
						C: 33,
						D: "not everything",
					},
				},
			},
			{
				name: "struct",
				value: func() interface{} {
					return &myStruct{
						E: &myStruct{
							a: "everything",
							B: "everything",
							C: 33,
							D: "not everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						E: &myStruct{
							a: "everything",
							B: expectedMask,
							C: 33,
							D: "not everything",
						},
					},
					withKeyRE: &myStruct{
						E: &myStruct{
							a: "everything",
							B: "everything",
							C: 33,
							D: "not everything",
						},
					},
					withBothRE: &myStruct{
						E: &myStruct{
							a: "everything",
							B: expectedMask,
							C: 33,
							D: "not everything",
						},
					},
					withBothDisabled: &myStruct{
						E: &myStruct{
							a: "everything",
							B: "everything",
							C: 33,
							D: "not everything",
						},
					},
				},
			},
			{
				name: "exported embedded struct pointer",
				value: func() interface{} {
					return &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							F: "everything",
							f: "everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							F: expectedMask,
							f: "everything",
						},
					},
					withKeyRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							F: "everything",
							f: "everything",
						},
					},
					withBothRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							F: expectedMask,
							f: "everything",
						},
					},
					withBothDisabled: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							F: "everything",
							f: "everything",
						},
					},
				},
			},
			{
				name: "exported embedded struct pointer",
				value: func() interface{} {
					return &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							Authorization: []string{randString, expectedMask, randString},
							authorization: []string{randString, "everything", randString},
							F:             expectedMask,
							f:             "everything",
						},
					},
					withKeyRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							Authorization: []string{expectedMask, expectedMask, expectedMask},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
					withBothRE: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							Authorization: []string{expectedMask, expectedMask, expectedMask},
							authorization: []string{randString, "everything", randString},
							F:             expectedMask,
							f:             "everything",
						},
					},
					withBothDisabled: &myStruct{
						EmbeddedStruct2: &EmbeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
				},
			},
			{
				name: "exported embedded struct",
				value: func() interface{} {
					return &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							F: "everything",
							f: "everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							F: expectedMask,
							f: "everything",
						},
					},
					withKeyRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							F: "everything",
							f: "everything",
						},
					},
					withBothRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							F: expectedMask,
							f: "everything",
						},
					},
					withBothDisabled: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							F: "everything",
							f: "everything",
						},
					},
				},
			},
			{
				name: "exported embedded struct",
				value: func() interface{} {
					return &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							Authorization: []string{randString, expectedMask, randString},
							authorization: []string{randString, "everything", randString},
							F:             expectedMask,
							f:             "everything",
						},
					},
					withKeyRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							Authorization: []string{expectedMask, expectedMask, expectedMask},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
					withBothRE: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							Authorization: []string{expectedMask, expectedMask, expectedMask},
							authorization: []string{randString, "everything", randString},
							F:             expectedMask,
							f:             "everything",
						},
					},
					withBothDisabled: &myStruct{
						EmbeddedStruct: EmbeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
				},
			},
			{
				name: "unexported embedded struct",
				value: func() interface{} {
					return &myStruct{
						embeddedStruct: embeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
						embeddedStruct2: &embeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myStruct{
						embeddedStruct: embeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
						embeddedStruct2: &embeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
					withKeyRE: &myStruct{
						embeddedStruct: embeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
						embeddedStruct2: &embeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
					withBothRE: &myStruct{
						embeddedStruct: embeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
						embeddedStruct2: &embeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
					withBothDisabled: &myStruct{
						embeddedStruct: embeddedStruct{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
						embeddedStruct2: &embeddedStruct2{
							Authorization: []string{randString, "everything", randString},
							authorization: []string{randString, "everything", randString},
							F:             "everything",
							f:             "everything",
						},
					},
				},
			},
			{
				name:  "custom scrub method",
				value: func() interface{} { return &myCustomScrubbedStruct{} },
				expected: expectedValues{
					withValueRE:      &myCustomScrubbedStruct{},
					withKeyRE:        &myCustomScrubbedStruct{},
					withBothRE:       &myCustomScrubbedStruct{},
					withBothDisabled: &myCustomScrubbedStruct{},
				},
			},
			{
				name:  "custom scrub method",
				value: func() interface{} { return &myCustomScrubbedStruct{unexported: "everything"} },
				expected: expectedValues{
					withValueRE:      &myCustomScrubbedStruct{unexported: expectedMask},
					withKeyRE:        &myCustomScrubbedStruct{unexported: "everything"},
					withBothRE:       &myCustomScrubbedStruct{unexported: expectedMask},
					withBothDisabled: &myCustomScrubbedStruct{unexported: "everything"},
				},
			},
			{
				name:  "custom scrub method",
				value: func() interface{} { return map[string]myCustomScrubbedStruct{"key": {unexported: "everything"}} },
				expected: expectedValues{
					withValueRE:      map[string]myCustomScrubbedStruct{"key": {unexported: expectedMask}},
					withKeyRE:        map[string]myCustomScrubbedStruct{"key": {unexported: "everything"}},
					withBothRE:       map[string]myCustomScrubbedStruct{"key": {unexported: expectedMask}},
					withBothDisabled: map[string]myCustomScrubbedStruct{"key": {unexported: "everything"}},
				},
			},
			{
				name:  "custom scrub method",
				value: func() interface{} { return []myCustomScrubbedStruct{{unexported: "everything"}, {unexported: "everything"}} },
				expected: expectedValues{
					withValueRE:      []myCustomScrubbedStruct{{unexported: expectedMask}, {unexported: expectedMask}},
					withKeyRE:        []myCustomScrubbedStruct{{unexported: "everything"}, {unexported: "everything"}},
					withBothRE:       []myCustomScrubbedStruct{{unexported: expectedMask}, {unexported: expectedMask}},
					withBothDisabled: []myCustomScrubbedStruct{{unexported: "everything"}, {unexported: "everything"}},
				},
			},
			{
				name: "custom scrub method",
				value: func() interface{} {
					return &myCustomScrubbedStruct{
						unexported: "everything",
						myStruct: &myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, "everything"},
							ApiKey:   9838923,
						},
					}
				},
				expected: expectedValues{
					withValueRE: &myCustomScrubbedStruct{
						unexported: expectedMask,
						myStruct: &myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, expectedMask},
							ApiKey:   9838923,
						},
					},
					withKeyRE: &myCustomScrubbedStruct{
						unexported: "everything",
						myStruct: &myStruct{
							PassWORD: expectedMask,
							password: randString,
							Secret_:  []string{expectedMask, expectedMask},
							ApiKey:   9838923,
						},
					},
					withBothRE: &myCustomScrubbedStruct{
						unexported: expectedMask,
						myStruct: &myStruct{
							PassWORD: expectedMask,
							password: randString,
							Secret_:  []string{expectedMask, expectedMask},
							ApiKey:   9838923,
						},
					},
					withBothDisabled: &myCustomScrubbedStruct{
						unexported: "everything",
						myStruct: &myStruct{
							PassWORD: randString,
							password: randString,
							Secret_:  []string{randString, "everything"},
							ApiKey:   9838923,
						},
					},
				},
			},
			{
				name: "json map type",
				value: func() interface{} {
					// equivalent to { "apikey": "...", "a": "...", "b": {}, "c": { "Password": "...", "d": "everything" }, "e": 33, "passwd": ["everything", "..."] }
					return map[string]interface{}{
						"apikey": randString,
						"a":      randString,
						"b":      map[string]interface{}{},
						"c": map[string]interface{}{
							"Password": randString,
							"d":        "everything",
						},
						"e":      33,
						"passwd": []interface{}{"everything", randString},
					}
				},
				expected: expectedValues{
					withValueRE: map[string]interface{}{
						"apikey": randString,
						"a":      randString,
						"b":      map[string]interface{}{},
						"c": map[string]interface{}{
							"Password": randString,
							"d":        expectedMask,
						},
						"e":      33,
						"passwd": []interface{}{expectedMask, randString},
					},
					withKeyRE: map[string]interface{}{
						"apikey": expectedMask,
						"a":      randString,
						"b":      map[string]interface{}{},
						"c": map[string]interface{}{
							"Password": expectedMask,
							"d":        "everything",
						},
						"e":      33,
						"passwd": []interface{}{expectedMask, expectedMask},
					},
					withBothRE: map[string]interface{}{
						"apikey": expectedMask,
						"a":      randString,
						"b":      map[string]interface{}{},
						"c": map[string]interface{}{
							"Password": expectedMask,
							"d":        expectedMask,
						},
						"e":      33,
						"passwd": []interface{}{expectedMask, expectedMask},
					},
					withBothDisabled: map[string]interface{}{
						"apikey": randString,
						"a":      randString,
						"b":      map[string]interface{}{},
						"c": map[string]interface{}{
							"Password": randString,
							"d":        "everything",
						},
						"e":      33,
						"passwd": []interface{}{"everything", randString},
					},
				},
			},
			{
				name: "json array type",
				value: func() interface{} {
					// equivalent to [ "everything", 33, [], {}, { "a": [ "everything" ], "password": "1234" }, null ]
					return []interface{}{
						"everything",
						33,
						[]interface{}{},
						map[string]interface{}{},
						map[string]interface{}{
							"a":        []interface{}{"everything"},
							"password": "1234",
						},
						nil,
					}
				},
				expected: expectedValues{
					withValueRE: []interface{}{
						expectedMask,
						33,
						[]interface{}{},
						map[string]interface{}{},
						map[string]interface{}{
							"a":        []interface{}{expectedMask},
							"password": "1234",
						},
						nil,
					},
					withKeyRE: []interface{}{
						"everything",
						33,
						[]interface{}{},
						map[string]interface{}{},
						map[string]interface{}{
							"a":        []interface{}{"everything"},
							"password": expectedMask,
						},
						nil,
					},
					withBothRE: []interface{}{
						expectedMask,
						33,
						[]interface{}{},
						map[string]interface{}{},
						map[string]interface{}{
							"a":        []interface{}{expectedMask},
							"password": expectedMask,
						},
						nil,
					},
					withBothDisabled: []interface{}{
						"everything",
						33,
						[]interface{}{},
						map[string]interface{}{},
						map[string]interface{}{
							"a":        []interface{}{"everything"},
							"password": "1234",
						},
						nil,
					},
				},
			},
		}

		for _, keyRE := range []string{ /* disabled */ "", keyRegexp} {
			keyRE := keyRE
			withKeyRE := keyRE != ``
			var state string
			if withKeyRE {
				state = "enabled"
			} else {
				state = "disabled"
			}
			name := fmt.Sprintf("with key regular expression %s", state)
			t.Run(name, func(t *testing.T) {
				for _, valueRE := range []string{ /* disabled */ "", valueRegexp} {
					valueRE := valueRE
					withValueRE := valueRE != ""
					var state string
					if withValueRE {
						state = "enabled"
					} else {
						state = "disabled"
					}
					name := fmt.Sprintf("with value regular expression %s", state)
					t.Run(name, func(t *testing.T) {
						s, err := sqsanitize.NewScrubber(keyRE, valueRE, expectedMask)
						require.NoError(t, err)

						for _, tc := range tests {
							tc := tc
							var value, expected interface{}
							t.Run(tc.name, func(t *testing.T) {
								switch v := tc.value.(type) {
								case string:
									// Need a string that can be set - hence using the address of the
									// local variable v
									value = &v
									// expected must have the *string type in order to have deep equal
									// working
									var want string
									if withKeyRE && withValueRE {
										want = tc.expected.withBothRE.(string)
									} else if withKeyRE {
										want = tc.expected.withKeyRE.(string)
									} else if withValueRE {
										want = tc.expected.withValueRE.(string)
									} else {
										want = tc.expected.withBothDisabled.(string)
									}
									expected = &want

								case func() interface{}:
									value = v()
									if withKeyRE && withValueRE {
										expected = tc.expected.withBothRE
									} else if withKeyRE {
										expected = tc.expected.withKeyRE
									} else if withValueRE {
										expected = tc.expected.withValueRE
									} else {
										expected = tc.expected.withBothDisabled
									}

								default:
									value = tc.value
									if withKeyRE && withValueRE {
										expected = tc.expected.withBothRE
									} else if withKeyRE {
										expected = tc.expected.withKeyRE
									} else if withValueRE {
										expected = tc.expected.withValueRE
									} else {
										expected = tc.expected.withBothDisabled
									}
								}
								info := sqsanitize.Info{}
								scrubbed, err := s.Scrub(value, info)
								require.NoError(t, err)
								require.Equal(t, expected, value)
								require.True(t, (scrubbed && len(info) > 0) || (!scrubbed && len(info) == 0))
							})
						}
					})
				}
			})
		}
	})

	t.Run("Usage", func(t *testing.T) {
		s, err := sqsanitize.NewScrubber("(?i)password", "forbidden", expectedMask)
		require.NoError(t, err)

		t.Run("URL Values", func(t *testing.T) {
			values := url.Values{
				"Password":   []string{"no", "pass", randString, "forbidden", randString},
				"_paSSwoRD ": []string{"no", "pass", randString, "forbidden", randString},
				fmt.Sprintf("%spassword%s", randString, randString): []string{"no", "pass", randString, "forbidden", randString},
				"other": []string{"forbidden", "whatforbidden", randString, "key"},
				"":      []string{"forbidden", "forbiddenwhat", randString, "key"},
			}
			info := sqsanitize.Info{}
			scrubbed, err := s.Scrub(values, info)
			require.NoError(t, err)
			require.True(t, scrubbed)

			expected := url.Values{
				"Password":   []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				"_paSSwoRD ": []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				fmt.Sprintf("%spassword%s", randString, randString): []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				"other": []string{expectedMask, "what" + expectedMask, randString, "key"},
				"":      []string{expectedMask, expectedMask + "what", randString, "key"},
			}
			require.Equal(t, expected, values)

			require.Contains(t, info, "no")
			require.Contains(t, info, "pass")
			require.Contains(t, info, randString)
			require.Contains(t, info, "forbidden")
			require.Contains(t, info, "whatforbidden")
			require.Contains(t, info, "forbiddenwhat")
		})

		t.Run("HTTP Request", func(t *testing.T) {
			s, err := sqsanitize.NewScrubber(`(?i)(passw(or)?d)|(secret)|(authorization)|(api_?key)|(access_?token)`, `(?:\d[ -]*?){13,16}`, expectedMask)
			require.NoError(t, err)

			t.Run("zero value", func(t *testing.T) {
				var req http.Request
				info := sqsanitize.Info{}
				scrubbed, err := s.Scrub(&req, info)
				require.NoError(t, err)
				require.False(t, scrubbed)
				require.Len(t, info, 0)
			})

			t.Run("random request", func(t *testing.T) {
				fuzzer := fuzz.New().NilChance(0).NumElements(10, 10)

				var url_ url.URL
				fuzzer.Fuzz(&url_)

				var headers, trailers http.Header
				fuzzer.Fuzz(&headers)
				fuzzer.Fuzz(&trailers)

				var host string
				fuzzer.Fuzz(&host)

				var form, postForm url.Values
				fuzzer.Fuzz(&form)
				fuzzer.Fuzz(&postForm)

				var multipartForm multipart.Form
				fuzzer.Fuzz(&multipartForm)

				// Insert some values forbidden by the regular expression
				postForm.Add("password", "1234")
				postForm.Add("password", "5678")
				messageFormat := "here is my credit card number %s."
				stringWithCreditCardNb := fmt.Sprintf(messageFormat, "4533-3432-3234-3334")
				form.Add("message", stringWithCreditCardNb)

				req := http.Request{
					Method:        "GET",
					URL:           &url_,
					Proto:         "HTTP/2",
					ProtoMajor:    2,
					ProtoMinor:    0,
					Header:        headers,
					ContentLength: 33,
					Host:          host,
					Form:          form,
					PostForm:      postForm,
					MultipartForm: &multipartForm,
					Trailer:       trailers,
					RemoteAddr:    "1.2.3.4",
					RequestURI:    url_.RequestURI(),
				}

				info := sqsanitize.Info{}
				scrubbed, err := s.Scrub(&req, info)
				require.NoError(t, err)
				require.True(t, scrubbed)

				// Check values were scrubbed
				require.Equal(t, []string{expectedMask, expectedMask}, req.PostForm["password"])
				require.Equal(t, []string{fmt.Sprintf(messageFormat, expectedMask)}, req.Form["message"])

				require.Contains(t, info, stringWithCreditCardNb)
				require.Contains(t, info, "1234")
				require.Contains(t, info, "5678")
			})
		})
	})
}

type EmbeddedStruct struct {
	Authorization []string // exported matching field
	authorization []string // unexported matching field
	F             string   // exported field
	f             string   // unexported field
}

// alternative names to avoid name collisions in the type definition
type EmbeddedStruct2 EmbeddedStruct
type embeddedStruct EmbeddedStruct
type embeddedStruct2 EmbeddedStruct

type myStruct struct {
	PassWORD         string      // matching exported string field
	Secret_          []string    // matching exported field with strings
	password         string      // unexported matching string field
	ApiKey           int         // non-string matching int field
	a                string      // unexported string value
	B                string      // exported string value
	C                int         // exported non-string
	D                string      // exported string value
	E                *myStruct   // recursive pointer
	Face             interface{} // exported interface value
	EmbeddedStruct               // exported embedded struct
	*EmbeddedStruct2             // exported embedded struct pointer
	embeddedStruct               // unexported embedded struct
	*embeddedStruct2             // unexported embedded struct pointer
}

type myCustomScrubbedStruct struct {
	unexported string // unexported field that will get scrubbed by the Scrub() method
	*myStruct
}

func (s *myCustomScrubbedStruct) Scrub(scrubber *sqsanitize.Scrubber, info sqsanitize.Info) (scrubbed bool, err error) {
	scrubbed, err = scrubber.Scrub(&s.unexported, info)
	if err != nil {
		return
	}
	var scrubbedMyStruct bool
	scrubbedMyStruct, err = scrubber.Scrub(s.myStruct, info)
	return scrubbed || scrubbedMyStruct, err
}

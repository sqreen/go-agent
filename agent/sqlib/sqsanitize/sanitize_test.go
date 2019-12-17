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

	//randString := testlib.RandUTF8String(512, 1024)
	randString := "toto"

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
				err = s.Scrub(&tc.value)
				require.NoError(t, err)
				require.Equal(t, tc.expected, tc.value)
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
			PassWORD         string    // matching exported string field
			Secret_          []string  // matching exported field with strings
			password         string    // unexported matching string field
			ApiKey           int       // non-string matching int field
			a                string    // unexported string value
			B                string    // exported string value
			C                int       // exported non-string
			D                string    // exported string value
			E                *myStruct // recursive pointer
			EmbeddedStruct             // exported embedded struct
			*EmbeddedStruct2           // exported embedded struct pointer
			embeddedStruct             // unexported embedded struct
			*embeddedStruct2           // unexported embedded struct pointer
		}

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
			//{ TODO
			//	name: "map of interface values",
			//	value: func() interface{} {
			//		// New local variable to avoid modifying &randString
			//		// This test will modify &myRandString (a *string value)
			//		//myRandString := randString
			//		return map[string]interface{}{
			//			//"passwd": &myRandString,
			//			"apikey": randString,
			//		}
			//	},
			//	expected: expectedValues{
			//		withValueRE: map[string]interface{}{
			//			//"passwd": &randString,
			//			"apikey": randString,
			//		},
			//		withKeyRE: map[string]interface{}{
			//			//"passwd": &expectedMask,
			//			"apikey": expectedMask,
			//		},
			//		withBothRE: map[string]interface{}{
			//			//"passwd": &expectedMask,
			//			"apikey": expectedMask,
			//		},
			//		withBothDisabled: map[string]interface{}{
			//			//"passwd": &randString,
			//			"apikey": randString,
			//		},
			//	},
			//},
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
						t.Parallel()

						s, err := sqsanitize.NewScrubber(keyRE, valueRE, expectedMask)
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
								err := s.Scrub(value)
								require.NoError(t, err)
								require.Equal(t, expected, value)
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
				fmt.Sprintf("%spassword%s", randString, randString): []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				"other": []string{"forbidden", "whatforbidden", randString, "key"},
				"":      []string{"forbidden", "forbiddenwhat", randString, "key"},
			}
			err := s.Scrub(values)
			require.NoError(t, err)
			expected := url.Values{
				"Password":   []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				"_paSSwoRD ": []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				fmt.Sprintf("%spassword%s", randString, randString): []string{expectedMask, expectedMask, expectedMask, expectedMask, expectedMask},
				"other": []string{expectedMask, "what" + expectedMask, randString, "key"},
				"":      []string{expectedMask, expectedMask + "what", randString, "key"},
			}
			require.Equal(t, expected, values)
		})
	})
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
	"gopkg.in/go-playground/assert.v1"
)

type contextWithMethods struct{}

func (c contextWithMethods) MyMethodField1() int {
	return 33
}
func (c *contextWithMethods) MyMethodField2() string {
	return "Sqreen"
}
func (c contextWithMethods) MyMethodField3() []bool {
	return []bool{true, true, false}
}

func TestBindingAccessor(t *testing.T) {
	for _, tc := range []struct {
		Title                    string
		Expression               string
		Context                  interface{}
		ExpectedValue            interface{}
		ExpectedExecutionError   interface{}
		ExpectedCompilationError bool
	}{
		{
			Title:      "nil value",
			Expression: `nil`,
			Context: struct {
				A string
				B int
			}{A: "Sqreen", B: 33},
			ExpectedValue: nil,
		},
		{
			Title:      "context value",
			Expression: `#`,
			Context: struct {
				A string
				B int
			}{A: "Sqreen", B: 33},
			ExpectedValue: struct {
				A string
				B int
			}{A: "Sqreen", B: 33},
		},
		{
			Title:         "array value",
			Expression:    `#[2]`,
			Context:       []int{37, 42, 23},
			ExpectedValue: 23,
		},
		{
			Title:         "map value",
			Expression:    `#['One']`,
			Context:       map[string]string{"One": "Sqreen"},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "function value",
			Expression: `#.A(#.B)`,
			Context: struct {
				A func(int) (int, error)
				B int
			}{A: func(i int) (int, error) { return i, nil }, B: 23},
			ExpectedValue: 23,
		},
		{
			Title:      "function value",
			Expression: `#.A(#.C.D).B`,
			Context: struct {
				A func(struct{ A, B, C, D string }) (struct{ A, B, C, D string }, error)
				B int
				C struct{ D struct{ A, B, C, D string } }
			}{A: func(d struct{ A, B, C, D string }) (struct{ A, B, C, D string }, error) { return d, nil }, B: 23, C: struct{ D struct{ A, B, C, D string } }{D: struct{ A, B, C, D string }{B: "yes"}}},
			ExpectedValue: "yes",
		},
		{
			Title:      "field value",
			Expression: `#.A`,
			Context: struct {
				A string
				B int
			}{A: "Sqreen", B: 33},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "field value",
			Expression: `#.B`,
			Context: struct {
				A string
				B int
			}{A: "Sqreen", B: 33},
			ExpectedValue: 33,
		},
		{
			Title:      "nested fields value",
			Expression: `#.A.B`,
			Context: struct {
				A struct{ B int }
				B int
			}{
				A: struct{ B int }{B: 42},
				B: 33,
			},
			ExpectedValue: 42,
		},
		{
			Title:      "pointer field traversal",
			Expression: `#.A.C`,
			Context: struct {
				A *struct{ C string }
				B int
			}{A: &struct{ C string }{C: "Sqreen"}, B: 33},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "nil pointer field traversal",
			Expression: `#.A.C`,
			Context: struct {
				A *struct{ C string }
				B int
			}{A: nil, B: 33},
			ExpectedExecutionError: true,
		},
		{
			Title:         "interface value",
			Expression:    `#.Face.Face`,
			Context:       struct{ Face interface{} }{struct{ Face interface{} }{"Sqreen"}},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "array value",
			Expression: `#.A[2]`,
			Context: struct {
				A []string
			}{A: []string{"Zero", "One", "Two"}},
			ExpectedValue: "Two",
		},
		{
			Title:         "map value index by an int",
			Expression:    `#.A[2780]`,
			Context:       struct{ A map[int]string }{A: map[int]string{2780: "Two"}},
			ExpectedValue: "Two",
		},
		{
			Title:         "map value index by a string",
			Expression:    `#.A['Two']`,
			Context:       struct{ A map[string]int }{A: map[string]int{"Two": 2}},
			ExpectedValue: 2,
		},
		{
			Title:         "non-existing map key gives a nil interface{} value",
			Expression:    `#.A['i dont exist']`,
			Context:       struct{ A map[string]uint16 }{},
			ExpectedValue: (interface{})(nil),
		},
		{
			Title:         "method",
			Expression:    `#.MyMethodField1`,
			Context:       contextWithMethods{},
			ExpectedValue: int(33),
		},
		{
			Title:         "method",
			Expression:    `#.MyMethodField2`,
			Context:       &contextWithMethods{},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "Nil pointer field access",
			Expression: `#.B.C`,
			Context: struct {
				A string
				B *struct{ C int }
			}{},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "method",
			Expression:             `#.MyMethodField2`,
			Context:                contextWithMethods{},
			ExpectedExecutionError: true,
		},
		{
			Title:         "method",
			Expression:    `#.MyMethodField3`,
			Context:       contextWithMethods{},
			ExpectedValue: []bool{true, true, false},
		},
		{
			Title:      "combination",
			Expression: `#.A.B[3].C[0].D['E']`,
			Context: struct {
				A struct {
					B map[int]struct {
						C []*struct{ D map[string]string }
					}
				}
			}{
				A: struct {
					B map[int]struct {
						C []*struct{ D map[string]string }
					}
				}{
					B: map[int]struct {
						C []*struct{ D map[string]string }
					}{
						3: {
							C: []*struct{ D map[string]string }{
								{
									D: map[string]string{"E": "Sqreen"},
								},
							},
						},
					},
				},
			},
			ExpectedValue: "Sqreen",
		},
		{
			Title:      "flat values transformation",
			Expression: "# | flat_values",
			Context: struct {
				A int
				B struct {
					C []interface{}
				}
			}{
				A: 33,
				B: struct{ C []interface{} }{
					C: []interface{}{
						1,
						struct{ D int }{D: 2},
						&struct{ E string }{E: "Sqreen"},
						map[interface{}]interface{}{
							"One":   1,
							2:       "Two",
							"Three": []int{27, 28},
						},
					},
				},
			},
			ExpectedValue: FlattenedResult{33, 1, 2, "Sqreen", 1, "Two", 27, 28},
		},
		{
			Title:      "flat keys transformation",
			Expression: "# | flat_keys",
			Context: struct {
				A int
				B struct {
					C []interface{}
				}
			}{
				A: 33,
				B: struct{ C []interface{} }{
					C: []interface{}{
						1,
						struct{ D int }{D: 2},
						&struct{ E string }{E: "Sqreen"},
						map[interface{}]interface{}{
							"One":   1,
							2:       "Two",
							"Three": []int{27, 28},
						},
					},
				},
			},
			ExpectedValue: FlattenedResult{"A", "B", "C", "D", "E", "One", 2, "Three"},
		},
		{
			Title:      "field value transformation",
			Expression: "#.B | flat_values",
			Context: struct {
				A int
				B struct {
					C []interface{}
				}
			}{
				A: 33,
				B: struct{ C []interface{} }{
					C: []interface{}{
						1,
						struct{ D int }{D: 2},
						&struct{ E string }{E: "Sqreen"},
						map[interface{}]interface{}{
							"One":   1,
							2:       "Two",
							"Three": []int{27, 28},
						},
					},
				},
			},
			ExpectedValue: FlattenedResult{1, 2, "Sqreen", 1, "Two", 27, 28},
		},

		//
		// Error cases
		//

		{
			Title:                    "empty binding accessor",
			Expression:               "",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               ".",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.",
			ExpectedCompilationError: true,
		},

		{
			Title:                    "syntax error",
			Expression:               "#..B",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.A[0",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.A[]",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.A[ ]",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.A['One",
			ExpectedCompilationError: true,
		},
		{
			Title:                    "syntax error",
			Expression:               "#.A | ",
			ExpectedCompilationError: true,
		},
		{
			Title:                  "field access to nil value",
			Expression:             "#.Foo",
			Context:                nil,
			ExpectedExecutionError: true,
		},
		{
			Title:                  "private field access",
			Expression:             "#.foo",
			Context:                struct{ foo string }{foo: "bar"},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "field access to non-struct value",
			Expression:             "#.Foo.Oops",
			Context:                struct{ Foo string }{Foo: "bar"},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "array index on a non-array value",
			Expression:             `#.Foo[1]`,
			Context:                struct{ Foo int }{Foo: 27},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "wrong map index type",
			Expression:             `#.Foo['One']`,
			Context:                struct{ Foo map[int]string }{Foo: map[int]string{1: "One"}},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "out of bounds array access",
			Expression:             `#.Foo[5]`,
			Context:                struct{ Foo []int }{Foo: []int{1, 2, 3}},
			ExpectedExecutionError: true,
		},
		{
			Title:                  "more than max execution depth",
			Expression:             `#[0][0][0][0][0][0][0][0][0][0][0]`,
			Context:                [][][][][][][][][][][]int{{{{{{{{{{{33}}}}}}}}}}},
			ExpectedExecutionError: bindingaccessor.ErrMaxExecutionDepth,
		},
		{
			Title:                  "less than max execution depth",
			Expression:             `#[0][0][0][0][0][0][0][0][0][0]`,
			Context:                [][][][][][][][][][][]int{{{{{{{{{{{33}}}}}}}}}}},
			ExpectedValue:          []int{33},
			ExpectedExecutionError: false,
		},
		{
			Title:      "more than max execution depth",
			Expression: `#.A.B.C.D.E.F.G.H.I.J.K.L`,
			Context: struct {
				A struct {
					B struct {
						C struct {
							D struct {
								E struct {
									F struct {
										G struct {
											H struct {
												I struct {
													J struct {
														K struct {
															L int
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}{},
			ExpectedExecutionError: bindingaccessor.ErrMaxExecutionDepth,
		},
	} {
		tc := tc
		t.Run(tc.Title, func(t *testing.T) {
			p, err := bindingaccessor.Compile(tc.Expression)
			if tc.ExpectedCompilationError {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			v, err := p(tc.Context)
			if tc.ExpectedExecutionError != nil {
				switch actual := tc.ExpectedExecutionError.(type) {
				case bool:
					if actual {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				case error:
					require.Error(t, err)
					xerrors.Is(err, actual)
				}
				return
			}

			if flatTransResult, ok := tc.ExpectedValue.(FlattenedResult); ok {
				requireEqualFlatResult(t, flatTransResult, v)
			} else {
				require.Equal(t, tc.ExpectedValue, v)
			}
		})
	}
}

func TestBindingAccessorUsage(t *testing.T) {
	t.Run("access to request values", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://sqreen.com/a/b/c?user=root&password=root", nil)

		req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 7.0; SM-G930VC Build/NRD90M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/58.0.3029.83 Mobile Safari/537.36")
		req.Header.Add("My-Header", "my value")
		req.Header.Add("My-Header", "my second value")
		req.Header.Add("My-Header", "my third value")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")

		type MyRequestWrapper struct {
			*http.Request //Access through embedding
			ClientIP      string
			Helper        struct {
				Query url.Values
			}
		}

		ctx := struct {
			Request MyRequestWrapper
		}{
			Request: MyRequestWrapper{
				Request:  req,
				ClientIP: "1.2.3.4",
				Helper:   struct{ Query url.Values }{Query: req.URL.Query()},
			},
		}

		for _, tc := range []struct {
			Expression    string
			ExpectedValue interface{}
		}{
			{
				Expression:    "#.Request.Method",
				ExpectedValue: ctx.Request.Method,
			},
			{
				Expression:    "#.Request.Proto",
				ExpectedValue: ctx.Request.Proto,
			},
			{
				Expression:    "#.Request.Host",
				ExpectedValue: ctx.Request.Host,
			},
			{
				Expression:    "#.Request.Header['User-Agent']",
				ExpectedValue: ctx.Request.Header["User-Agent"],
			},
			{
				Expression:    "#.Request.Header['I-Dont-Exist']",
				ExpectedValue: (interface{})(nil),
			},
			{
				Expression:    "#.Request.ClientIP",
				ExpectedValue: ctx.Request.ClientIP,
			},
			{
				Expression:    "#.Request.Header | flat_keys",
				ExpectedValue: FlattenedResult{"User-Agent", "My-Header", "Accept-Encoding"},
			},
			{
				Expression:    "#.Request.Header | flat_values",
				ExpectedValue: FlattenedResult{ctx.Request.Header.Get("User-Agent"), "my value", "my second value", "my third value", ctx.Request.Header.Get("Accept-Encoding")},
			},
			{
				Expression:    "#.Request.URL.Query | flat_values",
				ExpectedValue: FlattenedResult{"root", "root"},
			},
			{
				Expression:    "#.Request.URL.Query | flat_keys",
				ExpectedValue: FlattenedResult{"user", "password"},
			},
			{
				Expression:    "#.Request.Helper | flat_keys",
				ExpectedValue: FlattenedResult{"Query", "user", "password"},
			},
			{
				Expression:    "#.Request.Helper | flat_values",
				ExpectedValue: FlattenedResult{"root", "root"},
			},
			{
				Expression:    "#.Request.URL.RequestURI",
				ExpectedValue: "/a/b/c?user=root&password=root",
			},
		} {
			tc := tc
			t.Run(tc.Expression, func(t *testing.T) {
				p, err := bindingaccessor.Compile(tc.Expression)
				require.NoError(t, err)
				v, err := p(ctx)
				require.NoError(t, err)
				// Quick hack for transformations from maps that return an array that
				// cannot be compared to the expected value because the order of the map
				// accesses is not stable
				if flattened, ok := tc.ExpectedValue.(FlattenedResult); ok {
					requireEqualFlatResult(t, flattened, v)
				} else {
					require.Equal(t, tc.ExpectedValue, v)
				}
			})
		}
		require.NotNil(t, req)
	})
}

type FlattenedResult []interface{}

func requireEqualFlatResult(t *testing.T, expected FlattenedResult, value interface{}) {
	got := value.([]interface{})
	require.Equal(t, len(expected), len(got), got)
loop:
	for _, f := range expected {
		for _, g := range got {
			if assert.IsEqual(g, f) {
				continue loop
			}
		}
		require.Failf(t, "missing expected value", "expected `%v` having type `%T`", f, f)
	}
}

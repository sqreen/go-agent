// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

func TestFlatKeys(t *testing.T) {
	expectedKeys := func(count int) (r []interface{}) {
		keys := []interface{}{"F1", "F2", "F3", "F4"}
		for n := 0; n < count; n++ {
			r = append(r, keys...)
		}
		return
	}

	t.Run("basic values", func(t *testing.T) {
		type mytype struct {
			F1 byte
			F2 string
			F3 bool
			F4 int
		}

		t.Run("empty slice", func(t *testing.T) {
			v := []mytype{}
			out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("empty map", func(t *testing.T) {
			v := map[string]mytype{}
			out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("nil value", func(t *testing.T) {
			out := execFlatKeys(context.Background(), nil, newValueMaxDepth, NewValueMaxElements)
			require.Nil(t, out)
		})

		t.Run("slice", func(t *testing.T) {
			v := []mytype{
				{},
				{},
				{},
			}
			out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedKeys(3), out)
		})

		t.Run("array", func(t *testing.T) {
			v := [4]mytype{}
			out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedKeys(4), out)
		})

		t.Run("map", func(t *testing.T) {
			v := map[string]mytype{
				"k1": {},
				"k2": {},
			}
			out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			expected := append(expectedKeys(2), "k1", "k2")
			UnorderedEqual(t, expected, out)
		})
	})

	t.Run("composite value traversal", func(t *testing.T) {
		type mytype struct {
			F1 ***mytype         // pointer traversal
			F2 [][]mytype        // slice traversal
			F3 [2][2]*mytype     // array traversal
			F4 map[string]mytype // map traversal
		}
		ptr1 := &mytype{}
		ptr2 := &ptr1
		ptr3 := &ptr2
		v := mytype{
			F1: ptr3,
			F2: [][]mytype{
				{
					{}, // Empty value is enough for flat keys
					{}, // Empty value is enough for flat keys
				},
			},
			F3: [2][2]*mytype{
				{
					ptr1,
					ptr1,
				},
			},
			F4: map[string]mytype{
				"F1": {},
				"F2": {},
				"F3": {},
				"F4": {},
			},
		}
		out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		UnorderedEqual(t, expectedKeys(11), out)
	})

	t.Run("private struct fields only", func(t *testing.T) {
		v := struct {
			f1 byte
			f2 string
			f3 bool
			f4 int
		}{}
		out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		require.Nil(t, out)
	})

	t.Run("struct traversal of public fields only", func(t *testing.T) {
		type mytype struct {
			f1 byte
			F2 string
			f3 bool
			F4 int
		}
		w := mytype{
			f1: 32,
			F2: "sqreen",
			f3: true,
			F4: 33,
		}
		v := struct{ f1, F2, f3, F4 mytype }{w, w, w, w}
		out := execFlatKeys(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		UnorderedEqual(t, []interface{}{"F2", "F2", "F2", "F4", "F4", "F4"}, out)
	})

	t.Run("limits", func(t *testing.T) {
		in := url.Values{"": []string{"nokey"}, "both": []string{"y"}, "empty": []string{""}, "orphan": []string{""}, "prio": []string{"2"}, "z": []string{"post"}}
		allKeys := []interface{}{"", "both", "empty", "orphan", "prio", "z"}

		t.Run("less than max elements and max depth", func(t *testing.T) {
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, out, allKeys)
		})

		t.Run("more than max elements", func(t *testing.T) {
			in := url.Values{"": []string{"nokey"}, "both": []string{"y"}, "empty": []string{""}, "orphan": []string{""}, "prio": []string{"2"}, "z": []string{"post"}}
			maxElements := 3
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
			SliceContains(t, allKeys, out)
		})

		t.Run("less than max elements and max depth", func(t *testing.T) {
			in := map[string]interface{}{
				"k11": map[string]interface{}{
					"k21": map[string]interface{}{
						"k31": nil,
						"k32": nil,
						"k33": nil,
					},
					"k22": map[string]interface{}{
						"k31": nil,
						"k32": nil,
						"k33": nil,
					},
				},
				"k12": nil,
				"k13": nil,
			}
			out := execFlatKeys(context.Background(), in, 1, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, out, []interface{}{"k11", "k12", "k13"})
		})

		t.Run("more than max elements happening during a map entry traversal", func(t *testing.T) {
			in := map[string]map[string]struct{}{
				"k1": {
					"k11": {},
					"k12": {},
					"k13": {},
				},
				"k2": {
					"k21": {},
					"k22": {},
					"k23": {},
				},
				"k3": {
					"k21": {},
					"k22": {},
					"k23": {},
				},
			}
			maxElements := 10
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
		})

		t.Run("more than max elements happening during a struct traversal", func(t *testing.T) {
			in := struct{ F1, F2, F3 int }{}
			maxElements := 2
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
		})

		t.Run("more than max elements happening during a slice traversal", func(t *testing.T) {
			in := []map[string]struct{}{
				{
					"k1": {},
					"k2": {},
					"k3": {},
					"k4": {},
				},
			}
			maxElements := 2
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
		})

		t.Run("more than max depth", func(t *testing.T) {
			in := url.Values{"": []string{"nokey"}, "both": []string{"y"}, "empty": []string{""}, "orphan": []string{""}, "prio": []string{"2"}, "z": []string{"post"}}
			maxDepth := 2
			out := execFlatKeys(context.Background(), in, maxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, allKeys, out)
		})

		t.Run("more than max depth", func(t *testing.T) {
			var in [1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1]struct{ F1 string }
			out := execFlatKeys(context.Background(), in, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("more than max depth", func(t *testing.T) {
			in := struct {
				L0k0 string
				L0k1 int
				L0k2 bool
				L0k3 byte
				L0k4 struct {
					L1k0 string
					L1k1 int
					L1k2 bool
					L1k3 byte
					L1k4 struct {
						L2k0 string
						L2k1 int
						L2k2 bool
						L2k3 byte
						L2k4 uint
					}
				}
			}{}
			expectedKeys := func(level int) (r []interface{}) {
				r = make([]interface{}, 0, 5*level)
				for l := 0; l < level; l++ {
					for f := 0; f <= 4; f++ {
						r = append(r, fmt.Sprintf("L%dk%d", l, f))
					}
				}
				return
			}
			for maxDepth := 1; maxDepth <= 3; maxDepth++ {
				maxDepth := maxDepth
				t.Run(fmt.Sprintf("%d", maxDepth), func(t *testing.T) {
					out := execFlatKeys(context.Background(), in, maxDepth, NewValueMaxElements).([]interface{})
					UnorderedEqual(t, expectedKeys(maxDepth), out)
				})
			}
		})

		t.Run("both more than max depth and elements", func(t *testing.T) {
			t.Run("during a map entry traversal", func(t *testing.T) {
				in := map[string]interface{}{
					"k11": map[string]interface{}{
						"k21": map[string]interface{}{
							"k31": map[string]interface{}{
								"k41": map[string]interface{}{
									"k51": nil,
									"k52": nil,
									"k53": nil,
									"k54": nil,
									"k55": nil,
								},
							},
							"k32": nil,
							"k33": nil,
						},
						"k22": nil,
						"k23": nil,
						"k24": nil,
						"k25": nil,
					},
				}

				maxElements := 7
				maxDepth := 4
				out := execFlatKeys(context.Background(), in, maxDepth, maxElements).([]interface{})
				require.Len(t, out, maxElements)
				// Depending on the traversal (breadth vs depth), the actual values may defer
				SliceContains(t, []interface{}{"k11", "k21", "k31", "k41", "k32", "k33", "k22", "k23", "k24", "k25"}, out)
			})
		})
	})
}

func TestFlatValues(t *testing.T) {
	type mytype struct {
		F1 byte
		F2 string
		F3 bool
		F4 int
		F5 float64
	}

	// Create a random value of mytype
	var myValue mytype
	fuzz.New().Fuzz(&myValue)
	// helper function to create the set of expected values when they should appear `count` times.
	expectedValues := func(count int) (r []interface{}) {
		values := []interface{}{myValue.F1, myValue.F2, myValue.F3, myValue.F4, myValue.F5}
		for n := 0; n < count; n++ {
			r = append(r, values...)
		}
		return
	}
	expectedZeroValues := func(count int) (r []interface{}) {
		zero := mytype{}
		values := []interface{}{zero.F1, zero.F2, zero.F3, zero.F4, zero.F5}
		for n := 0; n < count; n++ {
			r = append(r, values...)
		}
		return
	}

	t.Run("basic values", func(t *testing.T) {
		t.Run("empty slice", func(t *testing.T) {
			v := []mytype{}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("empty map", func(t *testing.T) {
			v := map[string]mytype{}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("nil value", func(t *testing.T) {
			out := execFlatValues(context.Background(), nil, newValueMaxDepth, NewValueMaxElements)
			require.Nil(t, out)
		})

		t.Run("zero value", func(t *testing.T) {
			v := mytype{}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedZeroValues(1), out)
		})

		t.Run("pointer", func(t *testing.T) {
			v := &myValue
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedValues(1), out)
		})

		t.Run("nil pointer", func(t *testing.T) {
			var v *mytype
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Equal(t, []interface{}{(*mytype)(nil)}, out)
		})

		t.Run("slice", func(t *testing.T) {
			v := []mytype{
				myValue,
				myValue,
				myValue,
			}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedValues(3), out)
		})

		t.Run("array", func(t *testing.T) {
			v := [4]mytype{
				myValue,
				myValue,
				myValue,
				myValue,
			}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedValues(4), out)
		})

		t.Run("map", func(t *testing.T) {
			v := map[string]mytype{
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
			}
			out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, expectedValues(4), out)
		})
	})

	t.Run("composite value traversal", func(t *testing.T) {
		type myOtherType struct {
			F1 ***mytype         // pointer traversal
			F2 [][]mytype        // slice traversal
			F3 [2][2]mytype      // array traversal
			F4 map[string]mytype // map traversal
		}
		ptr1 := &myValue
		ptr2 := &ptr1
		ptr3 := &ptr2
		v := myOtherType{
			F1: ptr3,
			F2: [][]mytype{
				{
					myValue,
					myValue,
				},
			},
			F3: [2][2]mytype{
				{
					myValue,
					myValue,
				},
				// The other two will be zero values
			},
			F4: map[string]mytype{
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
			},
		}
		out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		UnorderedEqual(t, append(expectedValues(9), expectedZeroValues(2)...), out)
	})

	t.Run("private struct fields only", func(t *testing.T) {
		v := struct {
			f1 byte
			f2 string
			f3 bool
			f4 int
		}{}
		out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		require.Nil(t, out)
	})

	t.Run("struct traversal of public fields only", func(t *testing.T) {
		type mytype struct {
			f1 byte
			F2 string
			f3 bool
			F4 int
		}
		w := mytype{
			f1: 32,
			F2: "sqreen",
			f3: true,
			F4: 33,
		}
		v := struct{ f1, F2, f3, F4 mytype }{w, w, w, w}
		out := execFlatValues(context.Background(), v, newValueMaxDepth, NewValueMaxElements).([]interface{})
		UnorderedEqual(t, []interface{}{"sqreen", 33, "sqreen", 33}, out)
	})

	t.Run("limits", func(t *testing.T) {
		type myOtherType struct {
			F1 ***mytype         // pointer traversal
			F2 [][]mytype        // slice traversal
			F3 [2][2]mytype      // array traversal
			F4 map[string]mytype // map traversal
		}
		ptr1 := &myValue
		ptr2 := &ptr1
		ptr3 := &ptr2
		in := myOtherType{
			F1: ptr3,
			F2: [][]mytype{
				{
					myValue,
					myValue,
				},
			},
			F3: [2][2]mytype{
				{
					myValue,
					myValue,
				},
				// The other two will be zero values
			},
			F4: map[string]mytype{
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
				testlib.RandPrintableUSASCIIString(): myValue,
			},
		}
		allExpectedValues := append(expectedValues(9), expectedZeroValues(2)...)

		t.Run("less than max elements and max depth", func(t *testing.T) {
			out := execFlatValues(context.Background(), in, newValueMaxDepth, NewValueMaxElements).([]interface{})
			UnorderedEqual(t, allExpectedValues, out)
		})

		t.Run("more than max elements", func(t *testing.T) {
			maxElements := 3
			out := execFlatValues(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
			SliceContains(t, allExpectedValues, out)
		})

		t.Run("during a map entry traversal", func(t *testing.T) {
			in := map[string]map[string]mytype{
				"k1": {
					"k11": myValue,
					"k12": myValue,
					"k13": myValue,
				},
				"k2": {
					"k21": myValue,
					"k22": myValue,
					"k23": myValue,
				},
				"k3": {
					"k21": myValue,
					"k22": myValue,
					"k23": myValue,
				},
			}

			t.Run("more than max elements", func(t *testing.T) {
				maxElements := 10
				out := execFlatValues(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
				require.Len(t, out, maxElements)
			})

			t.Run("more than max elements", func(t *testing.T) {
				maxDepth := 1
				out := execFlatValues(context.Background(), in, maxDepth, NewValueMaxElements).([]interface{})
				require.Nil(t, out)
			})
		})

		t.Run("more than max elements happening during a struct traversal", func(t *testing.T) {
			in := struct{ F1, F2, F3 mytype }{myValue, myValue, myValue}
			maxElements := 2
			out := execFlatValues(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
		})

		t.Run("more than max elements happening during a slice traversal", func(t *testing.T) {
			in := []map[string]mytype{
				{
					"k1": myValue,
					"k2": myValue,
					"k3": myValue,
					"k4": myValue,
				},
			}
			maxElements := 2
			out := execFlatValues(context.Background(), in, newValueMaxDepth, maxElements).([]interface{})
			require.Len(t, out, maxElements)
		})

		t.Run("more than max depth", func(t *testing.T) {
			maxDepth := 2
			out := execFlatValues(context.Background(), in, maxDepth, NewValueMaxElements).([]interface{})
			require.Less(t, len(out), len(allExpectedValues))
			require.Less(t, len(out), NewValueMaxElements)
			SliceContains(t, allExpectedValues, out)
		})

		t.Run("more than max depth", func(t *testing.T) {
			var in [1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1][1]struct{ F1 string }
			out := execFlatValues(context.Background(), in, newValueMaxDepth, NewValueMaxElements).([]interface{})
			require.Nil(t, out)
		})

		t.Run("more than max depth", func(t *testing.T) {
			in := struct {
				L0k0 string
				L0k1 int
				L0k2 bool
				L0k3 byte
				L0k4 struct {
					L1k0 string
					L1k1 int
					L1k2 bool
					L1k3 byte
					L1k4 struct {
						L2k0 string
						L2k1 int
						L2k2 bool
						L2k3 byte
					}
				}
			}{}
			expectedValues := func(level int) (r []interface{}) {
				values := []interface{}{"", int(0), false, byte(0)}
				r = make([]interface{}, 0, 5*level)
				for l := 0; l < level; l++ {
					r = append(r, values...)
				}
				return
			}
			for maxDepth := 1; maxDepth <= 3; maxDepth++ {
				maxDepth := maxDepth
				t.Run(fmt.Sprintf("%d", maxDepth), func(t *testing.T) {
					out := execFlatValues(context.Background(), in, maxDepth, NewValueMaxElements).([]interface{})
					UnorderedEqual(t, expectedValues(maxDepth), out)
				})
			}
		})

		t.Run("both more than max depth and elements", func(t *testing.T) {
			t.Run("during a map entry traversal", func(t *testing.T) {
				in := []interface{}{
					[]interface{}{
						[]interface{}{
							[]interface{}{
								[]interface{}{
									"v51",
									"v52",
									"v53",
									"v54",
									"v55",
								},
							},
							"v31",
							"v32",
							"v33",
							"v34",
							"v35",
						},
						"v21",
						"v22",
						"v23",
						"v24",
						"v25",
					},
				}

				maxElements := 7
				maxDepth := 3
				out := execFlatValues(context.Background(), in, maxDepth, maxElements).([]interface{})
				require.Len(t, out, maxElements)
				// Depending on the traversal (breadth vs depth), the actual values may defer
				SliceContains(t, []interface{}{"v21", "v22", "v23", "v24", "v25", "v31", "v32", "v33", "v34", "v35"}, out)
			})
		})
	})
}

// UnorderedEqual checks that two arrays are equal no matter the order of
// their elements.
func UnorderedEqual(t *testing.T, expected []interface{}, got []interface{}) {
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

// SliceContains checks that the slice `slice` contains all elements of `contains`.
func SliceContains(t *testing.T, slice []interface{}, contains []interface{}) {
	require.LessOrEqual(t, len(contains), len(slice))
loop:
	for _, c := range contains {
		for _, s := range slice {
			if assert.IsEqual(c, s) {
				continue loop
			}
		}
		require.Failf(t, "missing expected value", "expected `%v` having type `%T`", c, c)
	}
}

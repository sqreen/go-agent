// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

//
//func TestNewAddSecurityHeadersCallbacks(t *testing.T) {
//	RunCallbackTest(t, TestConfig{
//		CallbacksCtor: callback.NewAddSecurityHeadersCallback,
//		ExpectProlog:  true,
//		PrologType:    reflect.TypeOf(callback.AddSecurityHeadersPrologCallbackType(nil)),
//		EpilogType:    reflect.TypeOf(callback.AddSecurityHeadersEpilogCallbackType(nil)),
//		InvalidTestCases: []interface{}{
//			nil,
//			33,
//			"yet another wrong type",
//			[]string{},
//			[]string{"one"},
//			[]string{"one", "two", "three"},
//			[]interface{}{[]string{"one", "two"}, []string{"three"}},
//			[]interface{}{[]string{"one", "two"}, []string{"three", "four"}, "nope"},
//		},
//		ValidTestCases: []ValidTestCase{
//			{
//				Rule: &RuleContextMockup{
//					config: []interface{}{
//						[]string{"k", "v"},
//						[]string{"one", "two"},
//						[]string{"canonical-header", "the value"},
//					},
//				},
//				TestCallback: func(t *testing.T, _ *RuleContextMockup, prolog sqhook.PrologCallback) {
//					expectedHeaders := http.Header{
//						"K":                []string{"v"},
//						"One":              []string{"two"},
//						"Canonical-Header": []string{"the value"},
//					}
//					actualProlog, ok := prolog.(callback.AddSecurityHeadersPrologCallbackType)
//					require.True(t, ok)
//					var rec http.ResponseWriter = httptest.NewRecorder()
//					epilog, err := actualProlog(&rec)
//					// Check it behaves as expected
//					require.NoError(t, err)
//					require.Equal(t, expectedHeaders, rec.Header())
//
//					// Test the epilog if any
//					if epilog != nil {
//						require.True(t, ok)
//						epilog(nil)
//					}
//				},
//			},
//		},
//	})
//}

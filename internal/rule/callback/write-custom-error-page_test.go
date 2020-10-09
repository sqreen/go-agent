// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

//func TestNewWriteCustomErrorPageCallbacks(t *testing.T) {
//	RunNativeCallbackTest(t, TestConfig{
//		CallbacksCtor: callback.NewWriteCustomErrorPageCallback,
//		ExpectProlog:  true,
//		PrologType:    reflect.TypeOf(callback.WriteCustomErrorPagePrologCallbackType(nil)),
//		EpilogType:    reflect.TypeOf(callback.WriteCustomErrorPageEpilogCallbackType(nil)),
//		InvalidTestCases: []interface{}{
//			33,
//			"yet another wrong type",
//		},
//		ValidTestCases: []ValidTestCase{
//			{
//				Rule:         &NativeRuleContextMockup{},
//				TestCallback: testWriteCustomErrorPageCallbacks(500),
//			},
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.CustomErrorPageRuleDataEntry{StatusCode: 33},
//				},
//				TestCallback: testWriteCustomErrorPageCallbacks(33),
//			},
//		},
//	})
//}

//func testWriteCustomErrorPageCallbacks(expectedStatusCode int) func(t *testing.T, rule *NativeRuleContextMockup, prolog sqhook.PrologCallback) {
//	return func(t *testing.T, _ *NativeRuleContextMockup, prolog sqhook.PrologCallback) {
//		actualProlog, ok := prolog.(callback.WriteCustomErrorPagePrologCallbackType)
//		require.True(t, ok)
//		var (
//			statusCode int
//			body       []byte
//		)
//
//		// Call the prolog callback
//		epilog, err := actualProlog(nil, nil, nil, &statusCode, &body)
//
//		// Check it behaves as expected
//		require.NoError(t, err)
//		require.Equal(t, expectedStatusCode, statusCode)
//		require.NotNil(t, body)
//
//		// Test the epilog if any
//		if epilog != nil {
//			epilog()
//		}
//	}
//}

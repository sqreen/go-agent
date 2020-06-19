// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

//func TestNewMonitorHTTPStatusCodeCallbacks(t *testing.T) {
//	RunNativeCallbackTest(t, TestConfig{
//		CallbacksCtor: callback.NewMonitorHTTPStatusCodeCallback,
//		ExpectProlog:  true,
//		PrologType:    reflect.TypeOf(callback.MonitorHTTPStatusCodePrologCallbackType(nil)),
//		EpilogType:    reflect.TypeOf(callback.MonitorHTTPStatusCodeEpilogCallbackType(nil)),
//		ValidTestCases: []ValidTestCase{
//			{
//				Rule: &RuleContextMockup{},
//				TestCallback: func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback) {
//					actualProlog, ok := prolog.(callback.MonitorHTTPStatusCodePrologCallbackType)
//					require.True(t, ok)
//					code := rand.Int()
//					rule.On("PushMetricsValue", code, uint64(1)).Return().Once()
//					epilog, err := actualProlog(nil, &code)
//					// Check it behaves as expected
//					require.NoError(t, err)
//
//					// Test the epilog if any
//					if epilog != nil {
//						require.True(t, ok)
//						epilog()
//					}
//				},
//			},
//		},
//	})
//}

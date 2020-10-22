// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

//
//func TestInAppWAFCallback(t *testing.T) {
//	if waf.Version() == nil {
//		t.SkipNow()
//	}
//
//	RunNativeCallbackTest(t, TestConfig{
//		CallbacksCtor: callback.NewWAFCallback,
//		ExpectProlog:  true,
//		InvalidTestCases: []interface{}{
//			33,
//			"yet another wrong type",
//			&NativeRuleContextMockup{},
//			// Binding accessor error
//			&NativeRuleContextMockup{
//				config: &api.WAFRuleDataEntry{
//					BindingAccessors: []string{
//						`#.Request.UserAgent`,
//					},
//					WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
//				},
//			},
//			// WAF Rule json error
//			&NativeRuleContextMockup{
//				config: &api.WAFRuleDataEntry{
//					BindingAccessors: []string{
//						`#.Request.UserAgent`,
//					},
//					WAFRules: `{"rules": [{"rule_id": "1",filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
//				},
//			},
//			// Empty list of binding accessors
//			&NativeRuleContextMockup{
//				config: &api.WAFRuleDataEntry{
//					BindingAccessors: []string{},
//					WAFRules:         `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
//				},
//			},
//			// Empty WAF Rule
//			&NativeRuleContextMockup{
//				config: &api.WAFRuleDataEntry{
//					BindingAccessors: []string{
//						`#.Request.UserAgent`,
//					},
//					WAFRules: `{"rules": []}`,
//				},
//			},
//		},
//		ValidTestCases: []ValidTestCase{
//			// -- Blocking Mode
//			// Block action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"Arachni"}},
//				}, sqhook.AbortError, true, true),
//				ExpectAbortedCallbackChain: true,
//			},
//			// Monitor action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"Arachni"}},
//				}, nil, true, true),
//			},
//			// No action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"go-http-client"}},
//				}, nil, false, true),
//			},
//			// -- Monitoring Mode
//			// Block action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"Arachni"}},
//				}, nil, true, false),
//			},
//			// Monitor action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"Arachni"}},
//				}, nil, true, false),
//			},
//			// No action
//			{
//				Rule: &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.UserAgent`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
//					},
//				},
//				TestCallback: testInAppWAFCallback(&http.Request{
//					Header: http.Header{"User-Agent": []string{"go-http-client"}},
//				}, nil, false, false),
//			},
//		},
//	})
//}
//
//func testInAppWAFCallback(req *http.Request, expectedErr error, shouldReportAttack, blockingMode bool) func(t *testing.T, rule *NativeRuleContextMockup, prolog sqhook.PrologCallback) {
//	return func(t *testing.T, rule *NativeRuleContextMockup, prolog sqhook.PrologCallback) {
//		actualProlog, ok := prolog.(callback.WAFPrologCallbackType)
//		require.True(t, ok)
//
//		// Store the request record into the request context
//		httpprotection.NewRequestContext(agentMockup, responseWriterMockup, requestReaderMockup, cancelHandlerContextFuncMockup)
//
//		// Prepare the test
//		rr.ExpectClientIP().Return(net.IP{1, 2, 3, 4})
//		rec := httptest.NewRecorder()
//		rule.ExpectBlockingMode().Return(blockingMode).Once()
//		if shouldReportAttack {
//			attack := &record.AttackEvent{}
//			rule.ExpectNewAttack(expectedErr == sqhook.AbortError && blockingMode, mock.Anything).Return(attack).Once()
//			rr.ExpectAddAttackEvent(attack).Once()
//		}
//
//		// Call the callback
//		var w http.ResponseWriter = rec
//		epilog, err := actualProlog(&w, &req)
//
//		// Check it behaves as expected
//		require.Equal(t, expectedErr, err)
//
//		// The in-app waf returned an abort
//		if err == sqhook.AbortError {
//			// It should respond with a bad request status
//			require.Equal(t, http.StatusBadRequest, rec.Code)
//			// It should return an epilog setting the return value in order to abort
//			// the request handling.
//			require.NotNil(t, epilog)
//			var err error
//			epilog(&err)
//			require.Error(t, err)
//		} else {
//			// No error => monitoring or nothing
//			if epilog != nil {
//				var err error
//				epilog(&err)
//				// No error expected.
//				require.NoError(t, err)
//			}
//		}
//	}
//}
//
//type RequestRecordMockup struct {
//	testmock.RequestRecordMockup
//}
//
//func (rr *RequestRecordMockup) AddAttackEvent(attack *record.AttackEvent) {
//	rr.Called(attack)
//}
//
//func (rr *RequestRecordMockup) ExpectAddAttackEvent(attack *record.AttackEvent) *mock.Call {
//	return rr.On("AddAttackEvent", attack)
//}
//
//func (rr *RequestRecordMockup) ClientIP() net.IP {
//	return rr.Called().Get(0).(net.IP)
//}
//
//func (rr *RequestRecordMockup) ExpectClientIP() *mock.Call {
//	return rr.On("ClientIP")
//}
//
//func BenchmarkWAF(b *testing.B) {
//	b.Run("callback time per request size", func(b *testing.B) {
//		// Benchmark a WAF rule going through every possible BAs we have.
//		// The input they read is increased so that we can observe the time and
//		// space threshold, but also how long it takes.
//		// Also covers issue SQR-8550 with large input values.
//
//		for size := 1; size <= 10000; size *= 10 {
//			nbElements := size
//			fuzzer := fuzz.New().NilChance(0).NumElements(nbElements, nbElements)
//			fuzzer.Funcs(
//				func(s *string, c fuzz.Continue) {
//					// Enforce nbElements UTF8 (gofuzz doesn't use the NumElements
//					// settings for strings)
//					*s = testlib.RandUTF8String(nbElements, nbElements)
//				},
//				func(v *map[string][]string, c fuzz.Continue) {
//					// Enforce one value per map entry here - the benchmark is too slow
//					// otherwise
//					m := make(map[string][]string, nbElements)
//					for n := 0; n < nbElements; n++ {
//						var key, value string
//						c.Fuzz(&key)
//						c.Fuzz(&value)
//						m[key] = []string{value}
//					}
//					*v = m
//				},
//			)
//
//			// Generate a request with variable-length inputs having nbElements
//			var (
//				headers, formValues map[string][]string
//				method, userAgent   string
//				requestURL          url.URL
//			)
//			// Form values accessed by #.Request.FilteredParams
//			fuzzer.Fuzz(&formValues)
//			// Headers accessed by #.Request.Header
//			fuzzer.Fuzz(&headers)
//			// Method accessed by #.Request.Method
//			fuzzer.Fuzz(&method)
//			// User-agent accessed by #.Request.UserAgent
//			fuzzer.Fuzz(&userAgent)
//			headers["User-Agent"] = []string{userAgent}
//			// Request URL accessed by #.Request.URL.RequestURI
//			fuzzer.Fuzz(&requestURL)
//
//			req := &http.Request{
//				Method: method,
//				Form:   formValues,
//				Header: headers,
//				URL:    &requestURL,
//			}
//
//			b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
//				wafRule := &NativeRuleContextMockup{
//					config: &api.WAFRuleDataEntry{
//						BindingAccessors: []string{
//							`#.Request.FilteredParams | flat_values`,
//							`#.Request.FilteredParams | flat_keys`,
//							`#.Request.Method`,
//							`#.Request.UserAgent`,
//							`#.Request.Header | flat_values`,
//							`#.Request.Header | flat_keys`,
//							`#.Request.Header['Dont exist']`,
//							`#.Request.URL.RequestURI`,
//						},
//						WAFRules: `{"rules": [{"rule_id": "rule_custom_552203d1f33ce0705f6c215f462199f1", "filters": [{"operator": "@rx", "targets": ["#.Request.FilteredParams | flat_values"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.FilteredParams | flat_keys"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.Method"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.Header | flat_values"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.Header | flat_keys"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.URL.RequestURI"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.Header['Dont exist']"], "transformations": [], "value": "oh my regular expression"}, {"operator": "@rx", "targets": ["#.Request.UserAgent"], "transformations": [], "value": "oh my regular expression"}]}], "flows": [{"name": "rs_728137e2322e1d7a692ca3099f08e831-blocking", "steps": [{"id": "start", "rule_ids": ["rule_custom_552203d1f33ce0705f6c215f462199f1"], "on_match": "exit_block"}]}]}`,
//					},
//				}
//				wafRule.ExpectBlockingMode().Return(true)
//
//				prolog, err := callback.NewWAFCallback(wafRule, nil)
//				require.NoError(b, err)
//				cb := prolog.(rule.CallbackObject)
//				defer cb.Close()
//				waf, ok := cb.Prolog().(callback.WAFPrologCallbackType)
//				require.True(b, ok)
//
//				// Request record
//				rr := &RequestRecordMockup{}
//				rr.ExpectClientIP().Return(net.IP{1, 2, 3, 4})
//				// Store the request record into the request context
//				ctx := context.WithValue(context.Background(), record.RequestRecordContextKey{}, rr)
//				req = req.WithContext(ctx)
//
//				var w http.ResponseWriter = httptest.NewRecorder()
//
//				b.ResetTimer()
//				b.ReportAllocs()
//				for n := 0; n < b.N; n++ {
//					_, err = waf(&w, &req)
//					if err != nil {
//						b.FailNow()
//					}
//				}
//			})
//		}
//	})
//}

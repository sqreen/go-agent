// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/sqreen/go-agent/tools/testlib/testmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInAppWAFCallback(t *testing.T) {
	RunCallbackTest(t, TestConfig{
		CallbacksCtor: callback.NewWAFCallback,
		ExpectProlog:  true,
		PrologType:    reflect.TypeOf(callback.WAFPrologCallbackType(nil)),
		EpilogType:    reflect.TypeOf(callback.WAFEpilogCallbackType(nil)),
		InvalidTestCases: []interface{}{
			33,
			"yet another wrong type",
			&RuleContextMockup{},
			// Binding accessor error
			&RuleContextMockup{
				config: &api.WAFRuleDataEntry{
					BindingAccessors: []string{
						`#.Request.'UserAgent`,
					},
					WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
				},
			},
			// WAF Rule json error
			&RuleContextMockup{
				config: &api.WAFRuleDataEntry{
					BindingAccessors: []string{
						`#.Request.UserAgent`,
					},
					WAFRules: `{"rules": [{"rule_id": "1",filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
				},
			},
			// Empty list of binding accessors
			&RuleContextMockup{
				config: &api.WAFRuleDataEntry{
					BindingAccessors: []string{},
					WAFRules:         `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
				},
			},
			// Empty WAF Rule
			&RuleContextMockup{
				config: &api.WAFRuleDataEntry{
					BindingAccessors: []string{
						`#.Request.UserAgent`,
					},
					WAFRules: `{"rules": []}`,
				},
			},
		},
		ValidTestCases: []ValidTestCase{
			// Block action
			{
				Rule: &RuleContextMockup{
					config: &api.WAFRuleDataEntry{
						BindingAccessors: []string{
							`#.Request.UserAgent`,
						},
						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_block"}]}]}`,
					},
				},
				TestCallback: testInAppWAFCallback(&http.Request{
					Header: http.Header{"User-Agent": []string{"Arachni"}},
				}, sqhook.AbortError, true),
				ExpectAbortedCallbackChain: true,
			},
			// Monitor action
			{
				Rule: &RuleContextMockup{
					config: &api.WAFRuleDataEntry{
						BindingAccessors: []string{
							`#.Request.UserAgent`,
						},
						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
					},
				},
				TestCallback: testInAppWAFCallback(&http.Request{
					Header: http.Header{"User-Agent": []string{"Arachni"}},
				}, nil, true),
			},
			// No action
			{
				Rule: &RuleContextMockup{
					config: &api.WAFRuleDataEntry{
						BindingAccessors: []string{
							`#.Request.UserAgent`,
						},
						WAFRules: `{"rules": [{"rule_id": "1","filters": [{"operator": "@rx","targets": ["#.Request.UserAgent"],"value": "Arachni"}]}],"flows": [{"name": "arachni_detection","steps": [{"id": "start","rule_ids": ["1"],"on_match": "exit_monitor"}]}]}`,
					},
				},
				TestCallback: testInAppWAFCallback(&http.Request{
					Header: http.Header{"User-Agent": []string{"go-http-client"}},
				}, nil, false),
			},
		},
	})
}

func testInAppWAFCallback(req *http.Request, expectedErr error, shouldReportAttack bool) func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback) {
	return func(t *testing.T, rule *RuleContextMockup, prolog sqhook.PrologCallback) {
		actualProlog, ok := prolog.(callback.WAFPrologCallbackType)
		require.True(t, ok)

		// Store the request record into the request context
		rr := &RequestRecordMockup{}
		ctx := context.WithValue(context.Background(), record.RequestRecordContextKey{}, rr)
		req = req.WithContext(ctx)

		// Prepare the test
		rr.ExpectClientIP().Return(net.IP{1, 2, 3, 4})
		rec := httptest.NewRecorder()
		var w http.ResponseWriter = rec
		if shouldReportAttack {
			attack := &record.AttackEvent{}
			rule.ExpectNewAttack(expectedErr == sqhook.AbortError, mock.Anything).Return(attack).Once()
			rr.ExpectAddAttackEvent(attack).Once()
		}

		// Call the callback
		epilog, err := actualProlog(&w, &req)

		// Check it behaves as expected
		require.Equal(t, expectedErr, err)

		// The in-app waf returned an abort
		if err == sqhook.AbortError {
			// It should respond with a bad request status
			require.Equal(t, http.StatusBadRequest, rec.Code)
			// It should return an epilog setting the return value in order to abort
			// the request handling.
			require.NotNil(t, epilog)
			var err error
			epilog(&err)
			require.Error(t, err)
		} else {
			// No error => monitoring or nothing
			if epilog != nil {
				var err error
				epilog(&err)
				// No error expected.
				require.NoError(t, err)
			}
		}
	}
}

type RequestRecordMockup struct {
	testmock.RequestRecordMockup
}

func (rr *RequestRecordMockup) AddAttackEvent(attack *record.AttackEvent) {
	rr.Called(attack)
}

func (rr *RequestRecordMockup) ExpectAddAttackEvent(attack *record.AttackEvent) *mock.Call {
	return rr.On("AddAttackEvent", attack)
}

func (rr *RequestRecordMockup) ClientIP() net.IP {
	return rr.Called().Get(0).(net.IP)
}

func (rr *RequestRecordMockup) ExpectClientIP() *mock.Call {
	return rr.On("ClientIP")
}

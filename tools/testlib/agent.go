package testlib

import (
	"net/http"
	"time"

	"github.com/sqreen/go-agent/agent/types"
	"github.com/stretchr/testify/mock"
)

type AgentMockup struct {
	mock.Mock
}

// Static assertion of correct interface implementation.
var _ types.Agent = &AgentMockup{}

func (a *AgentMockup) ResetExpectations() {
	a.Mock = mock.Mock{
		ExpectedCalls: a.Mock.ExpectedCalls,
	}
}

func (a *AgentMockup) NewRequestRecord(r *http.Request) types.RequestRecord {
	ret := a.Called(r)
	return ret.Get(0).(types.RequestRecord)
}

func (a *AgentMockup) ExpectNewRequestRecord(r interface{}) *mock.Call {
	return a.On("NewRequestRecord", r)
}

func (a *AgentMockup) GracefulStop() {
	a.Called()
}

func (a *AgentMockup) ExpectGracefulStop() *mock.Call {
	return a.On("GracefulStop")
}

func NewAgentForMiddlewareTestsWithoutSecurityResponse() (*AgentMockup, *HTTPRequestRecordMockup) {
	agent := &AgentMockup{}
	record := &HTTPRequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectSecurityResponse().Return(nil).Once()
	record.ExpectUserSecurityResponse().Return(nil).Maybe() // Some tests don't call it, such as those returning a handler error
	record.ExpectClose().Once()
	return agent, record
}

func NewAgentForMiddlewareTestsWithSecurityResponse(actionHandler http.Handler) (*AgentMockup, *HTTPRequestRecordMockup) {
	agent := &AgentMockup{}
	record := &HTTPRequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectSecurityResponse().Return(actionHandler).Once()
	record.ExpectClose().Once()
	return agent, record
}

func NewAgentForMiddlewareTestsWithUserSecurityResponse(actionHandler http.Handler) (*AgentMockup, *HTTPRequestRecordMockup) {
	agent := &AgentMockup{}
	record := &HTTPRequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectSecurityResponse().Return(nil).Once()
	record.ExpectUserSecurityResponse().Return(actionHandler)
	record.ExpectClose().Once()
	return agent, record
}

type HTTPRequestRecordMockup struct {
	mock.Mock
}

// Static assertion of correct interface implementation.
var _ types.RequestRecord = &HTTPRequestRecordMockup{}

func (r *HTTPRequestRecordMockup) NewCustomEvent(event string) types.CustomEvent {
	r.Called(event)
	return r
}

func (r *HTTPRequestRecordMockup) ExpectTrackEvent(event string) *mock.Call {
	return r.On("NewCustomEvent", event)
}

func (r *HTTPRequestRecordMockup) Close() {
	r.Called()
}

func (r *HTTPRequestRecordMockup) ExpectClose() *mock.Call {
	return r.On("Close")
}

func (r *HTTPRequestRecordMockup) NewUserAuth(id map[string]string, success bool) {
	r.Called(id, success)
}

func (r *HTTPRequestRecordMockup) ExpectTrackAuth(id map[string]string, success bool) *mock.Call {
	return r.On("NewUserAuth", id, success)
}

func (r *HTTPRequestRecordMockup) NewUserSignup(id map[string]string) {
	r.Called(id)
}

func (r *HTTPRequestRecordMockup) ExpectTrackSignup(id map[string]string) *mock.Call {
	return r.On("NewUserSignup", id)
}

func (r *HTTPRequestRecordMockup) Identify(id map[string]string) {
	r.Called(id)
}

func (r *HTTPRequestRecordMockup) SecurityResponse() http.Handler {
	ret := r.Called().Get(0)
	if ret == nil {
		return nil
	}
	return ret.(http.Handler)
}

func (r *HTTPRequestRecordMockup) ExpectSecurityResponse() *mock.Call {
	return r.On("SecurityResponse")
}

func (r *HTTPRequestRecordMockup) UserSecurityResponse() http.Handler {
	ret := r.Called().Get(0)
	if ret == nil {
		return nil
	}
	return ret.(http.Handler)
}

func (r *HTTPRequestRecordMockup) ExpectUserSecurityResponse() *mock.Call {
	return r.On("UserSecurityResponse")
}

func (r *HTTPRequestRecordMockup) ExpectIdentify(id map[string]string) *mock.Call {
	return r.On("Identify", id)
}

func (r *HTTPRequestRecordMockup) WithTimestamp(t time.Time) {
	r.Called(t)
}

func (r *HTTPRequestRecordMockup) ExpectWithTimestamp(t time.Time) *mock.Call {
	return r.On("WithTimestamp", t)
}

func (r *HTTPRequestRecordMockup) WithProperties(props types.EventProperties) {
	r.Called(props)
}

func (r *HTTPRequestRecordMockup) ExpectWithProperties(props types.EventProperties) *mock.Call {
	return r.On("WithProperties", props)
}

func (r *HTTPRequestRecordMockup) WithUserIdentifiers(id map[string]string) {
	r.Called(id)
}

func (r *HTTPRequestRecordMockup) ExpectWithUserIdentifiers(id map[string]string) *mock.Call {
	return r.On("WithUserIdentifiers", id)
}

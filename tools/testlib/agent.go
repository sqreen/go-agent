package testlib

import (
	"net/http"
	"time"

	"github.com/sqreen/go-agent/sdk"
	"github.com/stretchr/testify/mock"
)

type AgentMockup struct {
	mock.Mock
}

func (a *AgentMockup) ResetExpectations() {
	a.Mock = mock.Mock{
		ExpectedCalls: a.Mock.ExpectedCalls,
	}
}

func (a *AgentMockup) SecurityAction(req *http.Request) sdk.Action {
	ret := a.Called(req)
	if ret := ret.Get(0); ret != nil {
		return ret.(sdk.Action)
	}
	return nil
}

func (a *AgentMockup) ExpectSecurityAction(req *http.Request) *mock.Call {
	return a.On("SecurityAction", req)
}

func (a *AgentMockup) GracefulStop() {
	a.Called()
}

func (a *AgentMockup) ExpectGracefulStop() *mock.Call {
	return a.On("GracefulStop")
}

func (a *AgentMockup) NewRequestRecord(req *http.Request) sdk.RequestRecord {
	a.Called(req)
	return a
}

func (a *AgentMockup) ExpectNewRequestRecord(req *http.Request) *mock.Call {
	return a.On("NewRequestRecord", req)
}

func (a *AgentMockup) Close() {
	a.Called()
}

func (a *AgentMockup) ExpectClose() *mock.Call {
	return a.On("Close")
}

func (a *AgentMockup) NewCustomEvent(event string) sdk.CustomEvent {
	// Return itself as long as it can both implement RequestRecord and Event
	// interfaces without conflicting thanks to distinct method signatures.
	a.Called(event)
	return a
}

func (a *AgentMockup) ExpectTrackEvent(event string) *mock.Call {
	return a.On("NewCustomEvent", event)
}

func (a *AgentMockup) NewUserAuth(id map[string]string, success bool) {
	a.Called(id, success)
}

func (a *AgentMockup) ExpectTrackAuth(id map[string]string, success bool) *mock.Call {
	return a.On("NewUserAuth", id, success)
}

func (a *AgentMockup) NewUserSignup(id map[string]string) {
	a.Called(id)
}

func (a *AgentMockup) ExpectTrackSignup(id map[string]string) *mock.Call {
	return a.On("NewUserSignup", id)
}

func (a *AgentMockup) Identify(id map[string]string) {
	a.Called(id)
}

func (a *AgentMockup) ExpectIdentify(id map[string]string) *mock.Call {
	return a.On("Identify", id)
}

func (a *AgentMockup) WithTimestamp(t time.Time) {
	a.Called(t)
}

func (a *AgentMockup) ExpectWithTimestamp(t time.Time) *mock.Call {
	return a.On("WithTimestamp", t)
}

func (a *AgentMockup) WithProperties(props map[string]string) {
	a.Called(props)
}

func (a *AgentMockup) ExpectWithProperties(props map[string]string) *mock.Call {
	return a.On("WithProperties", props)
}

func (a *AgentMockup) WithUserIdentifiers(id map[string]string) {
	a.Called(id)
}

func (a *AgentMockup) ExpectWithUserIdentifiers(id map[string]string) *mock.Call {
	return a.On("WithUserIdentifiers", id)
}

func NewAgentForMiddlewareTests(req *http.Request) *AgentMockup {
	agent := &AgentMockup{}
	agent.ExpectNewRequestRecord(req).Once()
	agent.ExpectClose().Once()
	agent.ExpectSecurityAction(req).Return(nil).Once()
	return agent
}

type SecurityActionMockup struct {
	mock.Mock
	ApplyFunc func(w http.ResponseWriter)
}

func (m *SecurityActionMockup) Apply(w http.ResponseWriter) {
	m.Called(w)
	m.ApplyFunc(w)
}

func (m *SecurityActionMockup) ExpectApply() *mock.Call {
	return m.On("Apply", mock.Anything)
}

func NewSecurityActionBlockWithStatus(statusCode int) *SecurityActionMockup {
	action := &SecurityActionMockup{
		ApplyFunc: func(w http.ResponseWriter) {
			w.WriteHeader(statusCode)
		},
	}
	action.ExpectApply().Once()
	return action
}

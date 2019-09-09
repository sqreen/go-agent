// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package testlib

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/tools/testlib/testmock"
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

func (a *AgentMockup) NewRequestRecord(r *http.Request) (types.RequestRecord, *http.Request) {
	ret := a.Called(r)
	return ret.Get(0).(types.RequestRecord), r
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

func NewAgentForMiddlewareTestsWithoutSecurityResponse() (*AgentMockup, *testmock.RequestRecordMockup) {
	agent := &AgentMockup{}
	record := &testmock.RequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectWhitelisted().Return(false).Once()
	record.ExpectSecurityResponse().Return(nil).Once()
	record.ExpectUserSecurityResponse().Return(nil).Maybe() // Some tests don't call it, such as those returning a handler error
	record.ExpectClose().Once()
	return agent, record
}

func NewAgentForMiddlewareTestsWithSecurityResponse(actionHandler http.Handler) (*AgentMockup, *testmock.RequestRecordMockup) {
	agent := &AgentMockup{}
	record := &testmock.RequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectWhitelisted().Return(false).Once()
	record.ExpectSecurityResponse().Return(actionHandler).Once()
	record.ExpectClose().Once()
	return agent, record
}

func NewAgentForMiddlewareTestsWithUserSecurityResponse(actionHandler http.Handler) (*AgentMockup, *testmock.RequestRecordMockup) {
	agent := &AgentMockup{}
	record := &testmock.RequestRecordMockup{}
	agent.ExpectNewRequestRecord(mock.Anything).Return(record).Once()
	record.ExpectWhitelisted().Return(false).Once()
	record.ExpectSecurityResponse().Return(nil).Once()
	record.ExpectUserSecurityResponse().Return(actionHandler)
	record.ExpectClose().Once()
	return agent, record
}

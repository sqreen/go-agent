// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package mockups

import (
	"net"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/plog"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/stretchr/testify/mock"
)

type AgentMockup struct {
	mock.Mock
}

func (a *AgentMockup) FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error) {
	rets := a.Called(ip)
	if v := rets.Get(0); v != nil {
		action = v.(actor.Action)
	}
	exists = rets.Bool(1)
	err = rets.Error(2)
	return
}

func (a *AgentMockup) FindActionByUserID(userID map[string]string) (action actor.Action, exists bool) {
	rets := a.Called(userID)
	if v := rets.Get(0); v != nil {
		action = v.(actor.Action)
	}
	exists = rets.Bool(1)
	return
}

func (a *AgentMockup) Logger() *plog.Logger {
	if v := a.Called().Get(0); v != nil {
		return v.(*plog.Logger)
	}
	return nil
}

func (a *AgentMockup) Config() protectioncontext.ConfigReader {
	if v := a.Called().Get(0); v != nil {
		return v.(protectioncontext.ConfigReader)
	}
	return nil
}

func (a *AgentMockup) ExpectConfig() *mock.Call {
	return a.On("Config")
}

func (a *AgentMockup) SendClosedRequestContext(ctx protectioncontext.ClosedRequestContextFace) error {
	return a.Called(ctx).Error(0)
}

func (a *AgentMockup) ExpectSendClosedRequestContext(ctx interface{}) *mock.Call {
	return a.On("SendClosedRequestContext", ctx)
}

func (a *AgentMockup) IsIPAllowed(ip net.IP) bool {
	return a.Called(ip).Bool(0)
}

func (a *AgentMockup) ExpectIsIPAllowed(ip interface{}) *mock.Call {
	return a.On("IsIPAllowed", ip)
}

func (a *AgentMockup) IsPathAllowed(path string) bool {
	return a.Called(path).Bool(0)
}

func (a *AgentMockup) ExpectIsPathAllowed(path string) *mock.Call {
	return a.On("IsPathAllowed", path)
}

type AgentConfigMockup struct {
	mock.Mock
}

func (c *AgentConfigMockup) PrioritizedIPHeader() string {
	return ""
}

func (a *AgentConfigMockup) PrioritizedIPHeaderFormat() string {
	return ""
}

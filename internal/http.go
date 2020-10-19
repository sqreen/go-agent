// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"context"
	"net"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	http_protection_types "github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

// Root protection context
type RootHTTPProtectionContext struct {
	ctx           context.Context
	cancel        context.CancelFunc
	agent         *AgentType
	sqreenTime    sqtime.SharedStopWatch
	maxSqreenTime time.Duration
}

func NewRootHTTPProtectionContext(ctx context.Context) (*RootHTTPProtectionContext, context.CancelFunc) {
	agent := agentInstance.get()
	if agent == nil || !agent.isRunning() {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)

	return &RootHTTPProtectionContext{
		ctx:           ctx,
		cancel:        cancel,
		agent:         agent,
		maxSqreenTime: agent.performanceBudget,
	}, cancel
}

func (p *RootHTTPProtectionContext) SqreenTime() *sqtime.SharedStopWatch {
	return &p.sqreenTime
}

func (p *RootHTTPProtectionContext) DeadlineExceeded(needed time.Duration) (exceeded bool) {
	if p.maxSqreenTime <= 0 {
		// No max time duration
		return false
	}
	return p.sqreenTime.Duration()+needed >= p.maxSqreenTime
}

func (p *RootHTTPProtectionContext) Config() http_protection_types.ConfigReader {
	return p.agent.config
}

func (p *RootHTTPProtectionContext) FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error) {
	return p.agent.actors.FindIP(ip)
}

func (p *RootHTTPProtectionContext) FindActionByUserID(userID map[string]string) (action actor.Action, exists bool) {
	return p.agent.actors.FindUser(userID)
}

func (p *RootHTTPProtectionContext) IsIPAllowed(ip net.IP) (allowed bool) {
	allowed, matched, err := p.agent.actors.IsIPAllowed(ip)
	if err != nil {
		p.agent.logger.Error(sqerrors.Wrapf(err, "agent: unexpected error while searching `%s` into the ip passlist", ip))
	}
	if allowed {
		p.agent.addIPPasslistEvent(matched)
		p.agent.logger.Debugf("ip address `%s` matched the passlist entry `%s` and is allowed to pass through Sqreen monitoring and protections", ip, matched)
	}
	return allowed
}

func (p *RootHTTPProtectionContext) IsPathAllowed(path string) (allowed bool) {
	allowed = p.agent.actors.IsPathAllowed(path)
	if allowed {
		p.agent.addPathPasslistEvent(path)
		p.agent.logger.Debugf("request path `%s` found in the passlist and is allowed to pass through Sqreen monitoring and protections", path)
	}
	return allowed
}

func (p *RootHTTPProtectionContext) Context() context.Context {
	return p.ctx
}

func (p *RootHTTPProtectionContext) CancelContext() {
	p.cancel()
}

func (p *RootHTTPProtectionContext) Close(ctx http_protection_types.ClosedProtectionContextFace) {
	p.agent.sendClosedHTTPProtectionContext(ctx)
}

var _ http_protection_types.RootProtectionContext = (*RootHTTPProtectionContext)(nil)

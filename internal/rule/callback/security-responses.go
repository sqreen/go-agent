// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/backend/api"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// Event names
const (
	blockIPEventName      = "app.sqreen.action.block_ip"
	blockUserEventName    = "app.sqreen.action.block_user"
	redirectIPEventName   = "app.sqreen.action.redirect_ip"
	redirectUserEventName = "app.sqreen.action.redirect_user"
)

func NewIPSecurityResponseCallback(RuleFace, NativeCallbackConfig) (sqhook.PrologCallback, error) {
	return newIPSecurityResponsePrologCallback(), nil
}

type securityResponseError struct{}

func (securityResponseError) Error() string { return "aborted by a security response" }

func newIPSecurityResponsePrologCallback() httpprotection.BlockingPrologCallbackType {
	return func(m **httpprotection.RequestContext) (httpprotection.BlockingEpilogCallbackType, error) {
		ctx := *m
		ip := ctx.RequestReader.ClientIP()
		action, exists, err := ctx.FindActionByIP(ip)
		if err != nil {
			ctx.Logger().Error(err)
			return nil, nil
		}
		if !exists {
			return nil, nil
		}
		return func(e *error) {
			writeIPSecurityResponse(ctx, action, ip)
			*e = securityResponseError{}
		}, nil
	}
}

// The security responses are a bit weird as they allow to customize the
// response only for redirection, otherwise it blocks with the global blocking
// settings. The contract with the HTTP protection here is to use a distinct
// error value according to what we need.
func writeIPSecurityResponse(ctx *httpprotection.RequestContext, action actor.Action, ip net.IP) {
	var properties protectioncontext.EventProperties
	if redirect, ok := action.(actor.RedirectAction); ok {
		properties = newRedirectedIPEventProperties(redirect, ip)
		ctx.TrackEvent(redirectIPEventName).WithProperties(properties)
		writeRedirectionResponse(ctx.ResponseWriter, redirect.RedirectionURL())
	} else {
		properties = newBlockedIPEventProperties(action, ip)
		ctx.TrackEvent(blockIPEventName).WithProperties(properties)
		ctx.WriteDefaultBlockingResponse()
	}
}

func NewUserSecurityResponseCallback(RuleFace, NativeCallbackConfig) (sqhook.PrologCallback, error) {
	return newUserSecurityResponsePrologCallback(), nil
}

func newUserSecurityResponsePrologCallback() httpprotection.IdentifyUserPrologCallbackType {
	return func(m **httpprotection.RequestContext, uid *map[string]string) (httpprotection.BlockingEpilogCallbackType, error) {
		ctx := *m
		id := *uid
		action, exists := ctx.FindActionByUserID(id)
		if !exists {
			return nil, nil
		}
		return func(e *error) {
			writeUserSecurityResponse(ctx, action, id)
			*e = securityResponseError{}
		}, nil
	}
}

func writeUserSecurityResponse(ctx *httpprotection.RequestContext, action actor.Action, userID map[string]string) {
	// Since this call happens in the handler, we need to close its context
	// which also let know the HTTP protection layer that it shouldn't continue
	// with post-handler protections.
	defer ctx.CancelHandlerContext()
	var properties protectioncontext.EventProperties
	if redirect, ok := action.(actor.RedirectAction); ok {
		properties = newRedirectedUserEventProperties(redirect, userID)
		ctx.TrackEvent(redirectUserEventName).WithProperties(properties)
		writeRedirectionResponse(ctx.ResponseWriter, redirect.RedirectionURL())
	} else {
		properties = newBlockedUserEventProperties(action, userID)
		ctx.TrackEvent(blockUserEventName).WithProperties(properties)
		ctx.WriteDefaultBlockingResponse()
	}
}

func writeRedirectionResponse(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusSeeOther)
}

// blockedIPEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type blockedIPEventProperties struct {
	action actor.Action
	ip     net.IP
}

func newBlockedIPEventProperties(action actor.Action, ip net.IP) *blockedIPEventProperties {
	return &blockedIPEventProperties{
		action: action,
		ip:     ip,
	}
}
func (p *blockedIPEventProperties) MarshalJSON() ([]byte, error) {
	pb := api.NewBlockedIPEventPropertiesFromFace(p)
	return json.Marshal(pb)
}
func (p *blockedIPEventProperties) GetActionId() string {
	return p.action.ActionID()
}
func (p *blockedIPEventProperties) GetOutput() api.BlockedIPEventProperties_Output {
	return *api.NewBlockedIPEventProperties_OutputFromFace(p)
}
func (p *blockedIPEventProperties) GetIpAddress() string {
	return p.ip.String()
}

// blockedUserEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type blockedUserEventProperties struct {
	action actor.Action
	userID map[string]string
}

func newBlockedUserEventProperties(action actor.Action, userID map[string]string) *blockedUserEventProperties {
	return &blockedUserEventProperties{
		action: action,
		userID: userID,
	}
}
func (p *blockedUserEventProperties) MarshalJSON() ([]byte, error) {
	pb := api.NewBlockedUserEventPropertiesFromFace(p)
	return json.Marshal(pb)
}
func (p *blockedUserEventProperties) GetActionId() string {
	return p.action.ActionID()
}
func (p *blockedUserEventProperties) GetOutput() api.BlockedUserEventPropertiesOutput {
	return *api.NewBlockedUserEventPropertiesOutputFromFace(p)
}
func (p *blockedUserEventProperties) GetUser() map[string]string {
	return p.userID
}

// redirectedUserEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type redirectedUserEventProperties struct {
	action actor.Action
	userID map[string]string
}

func newRedirectedUserEventProperties(action actor.Action, userID map[string]string) *redirectedUserEventProperties {
	return &redirectedUserEventProperties{
		action: action,
		userID: userID,
	}
}
func (p *redirectedUserEventProperties) MarshalJSON() ([]byte, error) {
	pb := api.NewRedirectedUserEventPropertiesFromFace(p)
	return json.Marshal(pb)
}
func (p *redirectedUserEventProperties) GetActionId() string {
	return p.action.ActionID()
}
func (p *redirectedUserEventProperties) GetOutput() api.RedirectedUserEventPropertiesOutput {
	return *api.NewRedirectedUserEventPropertiesOutputFromFace(p)
}
func (p *redirectedUserEventProperties) GetUser() map[string]string {
	return p.userID
}

// redirectedIPEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type redirectedIPEventProperties struct {
	action actor.RedirectAction
	ip     net.IP
}

func newRedirectedIPEventProperties(action actor.RedirectAction, ip net.IP) *redirectedIPEventProperties {
	return &redirectedIPEventProperties{
		action: action,
		ip:     ip,
	}
}
func (p *redirectedIPEventProperties) MarshalJSON() ([]byte, error) {
	pb := api.NewRedirectedIPEventPropertiesFromFace(p)
	return json.Marshal(pb)
}
func (p *redirectedIPEventProperties) GetActionId() string {
	return p.action.ActionID()
}
func (p *redirectedIPEventProperties) GetOutput() api.RedirectedIPEventPropertiesOutput {
	return *api.NewRedirectedIPEventPropertiesOutputFromFace(p)
}
func (p *redirectedIPEventProperties) GetIpAddress() string {
	return p.ip.String()
}
func (p *redirectedIPEventProperties) GetURL() string {
	return p.action.RedirectionURL()
}

// SecurityResponseMatch is an error type wrapping the security response that
// matched the request and helping in bubbling up to Sqreen's middleware
// function to abort the request.
type SecurityResponseMatch struct{}

func (SecurityResponseMatch) Error() string {
	return "a security response matched the request"
}

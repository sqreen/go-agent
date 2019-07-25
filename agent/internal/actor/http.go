// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package actor

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/httphandler"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
)

// Event names.
const (
	blockIPEventName      = "app.sqreen.action.block_ip"
	blockUserEventName    = "app.sqreen.action.block_user"
	redirectIPEventName   = "app.sqreen.action.redirect_ip"
	redirectUserEventName = "app.sqreen.action.redirect_user"
)

// NewIPActionHTTPHandler returns a HTTP handler that should be applied at the
// request handler level to perform the security response.
func NewIPActionHTTPHandler(action Action, ip net.IP) (http.Handler, error) {
	switch actual := action.(type) {
	case *timedAction:
		return NewIPActionHTTPHandler(actual.Action, ip)
	case blockAction:
		return newBlockHTTPHandler(blockIPEventName, newBlockedIPEventProperties(actual, ip)), nil
	case *redirectAction:
		return newRedirectHTTPHandler(redirectIPEventName, newRedirectedIPEventProperties(actual, ip), actual.URL), nil
	}
	return nil, sqerrors.Errorf("unexpected IP action type `%T`", action)
}

// NewUserActionHTTPHandler returns a HTTP handler that should be applied at
// the request handler level to perform the security response.
func NewUserActionHTTPHandler(action Action, userID map[string]string) (http.Handler, error) {
	switch actual := action.(type) {
	case *timedAction:
		return NewUserActionHTTPHandler(actual.Action, userID)
	case blockAction:
		return newBlockHTTPHandler(blockUserEventName, newBlockedUserEventProperties(actual, userID)), nil
	case *redirectAction:
		return newRedirectHTTPHandler(redirectUserEventName, newRedirectedUserEventProperties(actual, userID), actual.URL), nil
	}
	return nil, sqerrors.Errorf("unexpected user action type `%T`", action)
}

// blockHTTPHandler implements the http.Handler interface and holds the event
// data corresponding to the action.
type blockHTTPHandler struct {
	eventName       string
	eventProperties types.EventProperties
}

// Static assertion that http.Handler is implemented.
var _ http.Handler = &blockHTTPHandler{}

func newBlockHTTPHandler(eventName string, properties types.EventProperties) *blockHTTPHandler {
	return &blockHTTPHandler{
		eventName:       eventName,
		eventProperties: properties,
	}
}

// ServeHTTP writes the HTTP status code 500 into the HTTP response writer `w`.
// The caller needs to abort the request.
func (a *blockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	record := sdk.FromContext(r.Context())
	record.TrackEvent(a.eventName).WithProperties(a.eventProperties)
	httphandler.WriteResponse(w, r, nil, 500, nil)
}

// blockedIPEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type blockedIPEventProperties struct {
	action Action
	ip     net.IP
}

// Static assert that blockedIPEventProperties implements types.EventProperties.
var _ types.EventProperties = &blockedIPEventProperties{}

func newBlockedIPEventProperties(action Action, ip net.IP) *blockedIPEventProperties {
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
	action Action
	userID map[string]string
}

// Static assert that blockedUserEventProperties implements
// `types.EventProperties`.
var _ types.EventProperties = &blockedUserEventProperties{}

func newBlockedUserEventProperties(action Action, userID map[string]string) *blockedUserEventProperties {
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
	action Action
	userID map[string]string
}

// Static assert that redirectedUserEventProperties implements `types.EventProperties`.
var _ types.EventProperties = &redirectedUserEventProperties{}

func newRedirectedUserEventProperties(action Action, userID map[string]string) *redirectedUserEventProperties {
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

// redirectHTTPHandler implements the http.Handler interface and holds the event
// data corresponding to the action.
type redirectHTTPHandler struct {
	eventName       string
	eventProperties types.EventProperties
	location        string
}

// Static assertion that http.Handler is implemented.
var _ http.Handler = &redirectHTTPHandler{}

func newRedirectHTTPHandler(eventName string, properties types.EventProperties, location string) *redirectHTTPHandler {
	return &redirectHTTPHandler{
		eventName:       eventName,
		eventProperties: properties,
		location:        location,
	}
}

// ServeHTTP writes the HTTP status code 500 into the HTTP response writer `w`.
// The caller needs to abort the request.
func (a *redirectHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	record := sdk.FromContext(r.Context())
	record.TrackEvent(a.eventName).WithProperties(a.eventProperties)
	w.Header().Set("Location", a.location)
	w.WriteHeader(http.StatusSeeOther)
}

// redirectedIPEventProperties implements `types.EventProperties` to be marshaled
// to an SDK event property structure.
type redirectedIPEventProperties struct {
	action *redirectAction
	ip     net.IP
}

// Static assert that blockedIPEventProperties implements types.EventProperties.
var _ types.EventProperties = &redirectedIPEventProperties{}

func newRedirectedIPEventProperties(action *redirectAction, ip net.IP) *blockedIPEventProperties {
	return &blockedIPEventProperties{
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
	return p.action.URL
}

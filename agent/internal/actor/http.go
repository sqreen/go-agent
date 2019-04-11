// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Security Action HTTP Handlers
//
// Constructors `NewUserActionHTTPHandler()` and `NewIPActionHTTPHandler()`
// allow to create a `http.Handler` from an action that matched a user or an IP
// address. They allow to apply the expected security response to the request's
// response. The user and IP address are used as properties of events performed
// by handlers.
package actor

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
)

// Event names.
const (
	blockIPEventName   = "app.sqreen.action.block_ip"
	blockUserEventName = "app.sqreen.action.block_user"
)

// NewIPActionHTTPHandler returns a HTTP handler that should be applied at the
// request handler level to perform the security response.
func NewIPActionHTTPHandler(action Action, ip net.IP) http.Handler {
	return newBlockHTTPHandler(blockIPEventName, newBlockedIPEventProperties(action, ip))
}

// NewUserActionHTTPHandler returns a HTTP handler that should be applied at
// the request handler level to perform the security response.
func NewUserActionHTTPHandler(action Action, userID map[string]string) http.Handler {
	return newBlockHTTPHandler(blockUserEventName, newBlockedUserEventProperties(action, userID))
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
	w.WriteHeader(500)
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

// blockedIPEventProperties implements `types.EventProperties` to be marshaled
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
func (p *blockedUserEventProperties) GetOutput() api.BlockedUserEventProperties_Output {
	return *api.NewBlockedUserEventProperties_OutputFromFace(p)
}
func (p *blockedUserEventProperties) GetUser() map[string]string {
	return p.userID
}

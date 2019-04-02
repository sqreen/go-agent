// Copyright 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

/*

Security Actions

Security actions are stored using structures implementing the Action interface.
Actions can have a time duration by implementing the Timed interface.

Security action HTTP handlers

NewActionHandler() allows to get a http.Handler that applies the security
response (eg. blocking). If required, security response HTTP handlers can use
the SDK.

*/
package actor

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
)

// Action kinds.
const (
	actionKind_BlockIP = "block_ip"
)

// Event names.
const (
	blockIPEventName = "app.sqreen.action.block_ip"
)

// NewActionHandler returns a http.Handler for the given security action and IP
// address. This HTTP handler can be applied at the request handler level to
// perform the security response.
func NewActionHandler(action Action, ip net.IP) http.Handler {
	switch actual := action.(type) {
	case *blockIPAction:
		return newBlockIPActionHandler(actual, ip)
	}
	return nil
}

// Action is an interface common to each concrete action type stored in the data
// structures, and allowing to type-switch the stored values.
type Action interface {
	ActionID() string
}

// Timed is an interface implemented by actions having an expiration time.
type Timed interface {
	Expired() bool
}

// blockIPAction is an action type blocking the matching IP address.
type blockIPAction struct {
	ID string
}

// timedBlockIPAction is a blockIPAction with a time deadline after which it is
// considered expired.
type timedBlockIPAction struct {
	Action
	deadline time.Time
}

func newBlockIPAction(id string) *blockIPAction {
	return &blockIPAction{
		ID: id,
	}
}

func (a *blockIPAction) ActionID() string {
	return a.ID
}

// withDuration sets a time duration to an action. The returned value implements
// the Action and Timed interfaces.
func withDuration(action Action, duration time.Duration) *timedBlockIPAction {
	return &timedBlockIPAction{
		Action:   action,
		deadline: time.Now().Add(duration),
	}
}

// Expired is true when the deadline has expired, false otherwise.
func (a *timedBlockIPAction) Expired() bool {
	// Is the current time after the deadline?
	return time.Now().After(a.deadline)
}

// blockIPActionHandler is the blocking HTTP handler action to be applied to the
// request.
type blockIPActionHandler struct {
	*blockIPAction
	ip net.IP
}

func newBlockIPActionHandler(action *blockIPAction, ip net.IP) *blockIPActionHandler {
	return &blockIPActionHandler{
		blockIPAction: action,
		ip:            ip,
	}
}

// ServeHTTP writes the HTTP status code 500 into the HTTP response writer `w`.
// The caller needs to abort the request.
func (a *blockIPActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	record := sdk.FromContext(r.Context())
	record.TrackEvent(blockIPEventName).WithProperties((*blockedIPEventProperties)(a))
	w.WriteHeader(500)
}

// blockedIPEventProperties is an adapter type. It implements
// `types.EventProperties` to be marshaled to an SDK event property structure.
type blockedIPEventProperties blockIPActionHandler

// Static assert that blockedIPEventProperties implements types.EventProperties.
var _ types.EventProperties = &blockedIPEventProperties{}

// Static assert that blockedIPEventProperties implements
// api.BlockedIPEventPropertiesFace.
var _ api.BlockedIPEventPropertiesFace = &blockedIPEventProperties{}

func (p *blockedIPEventProperties) MarshalJSON() ([]byte, error) {
	pb := api.NewBlockedIPEventPropertiesFromFace(p)
	return json.Marshal(pb)
}

func (p *blockedIPEventProperties) GetActionId() string {
	return p.blockIPAction.ID
}

func (p *blockedIPEventProperties) GetOutput() api.BlockedIPEventProperties_Output {
	return *api.NewBlockedIPEventProperties_OutputFromFace(p)
}

func (p *blockedIPEventProperties) GetIpAddress() string {
	return p.ip.String()
}

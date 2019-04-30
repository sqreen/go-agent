// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

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

const sqreenBlockPage = `<!DOCTYPE html><html lang="en"><head> <meta charset="UTF-8"> <meta name="viewport" content="width=device-width, initial-scale=1.0"> <meta http-equiv="X-UA-Compatible" content="ie=edge"> <title>Sqreen has detected an attack.</title> <style>html, body, div, span, h1, a{margin: 0; padding: 0; border: 0; font-size: 100%; font: inherit; vertical-align: baseline}body{background: -webkit-radial-gradient(26% 19%, circle, #fff, #f4f7f9); background: radial-gradient(circle at 26% 19%, #fff, #f4f7f9); display: -webkit-box; display: -ms-flexbox; display: flex; -webkit-box-pack: center; -ms-flex-pack: center; justify-content: center; -webkit-box-align: center; -ms-flex-align: center; align-items: center; -ms-flex-line-pack: center; align-content: center; width: 100%; min-height: 100vh; line-height: 1}svg, h1, p{display: block}svg{margin: 0 auto 4vh}h1{font-family: sans-serif; font-weight: 300; font-size: 34px; color: #384886; line-height: normal}p{font-size: 18px; line-height: normal; color: #b8bccc; font-family: sans-serif; font-weight: 300}a{color: #b8bccc}.flex{text-align: center}</style></head><body> <div class="flex"> <svg xmlns="http://www.w3.org/2000/svg" width="230" height="250" viewBox="0 0 230 250" enable-background="new 0 0 230 250"> <style>.st0{opacity: 0.4; filter: url(#a);}.st1{fill: #FFFFFF;}.st2{fill: #B0ACFF;}.st3{fill: #4842B7;}.st4{fill: #1E0936;}</style> <filter id="a" width="151.7%" height="146%" x="-25.8%" y="-16%" filterUnits="objectBoundingBox"> <feOffset dy="14" in="SourceAlpha" result="shadowOffsetOuter1"/> <feGaussianBlur in="shadowOffsetOuter1" result="shadowBlurOuter1" stdDeviation="13"/> <feColorMatrix in="shadowBlurOuter1" values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.05 0"/> </filter> <g class="st0"> <path id="b_2_" d="M202.6 34.9c-.2-1.2-.8-2.1-1.9-2.8-3.8-2-37.9-20.1-85.7-20.1-48.8 0-84.2 19.3-85.7 20.1-1 .6-1.6 1.6-1.8 2.7-14.8 123.2 84.7 176.3 85.7 176.8.6.3 1.2.4 1.8.4.6 0 1.2-.1 1.7-.4 1-.5 100.4-55 85.9-176.7z"/> </g> <path id="b_1_" d="M202.6 34.9c-.2-1.2-.8-2.1-1.9-2.8-3.8-2-37.9-20.1-85.7-20.1-48.8 0-84.2 19.3-85.7 20.1-1 .6-1.6 1.6-1.8 2.7-14.8 123.2 84.7 176.3 85.7 176.8.6.3 1.2.4 1.8.4.6 0 1.2-.1 1.7-.4 1-.5 100.4-55 85.9-176.7z" class="st1"/> <g id="nest-cmyk-indigo"> <ellipse id="sqreen" cx="115.5" cy="69.9" class="st2" rx="12.7" ry="12.7"/> <path id="app" d="M113.6 91.9V71.5L95.5 61.1v18l6.4-3.7c.5 1.1 1 2.2 1.7 3.2L97 82.3l16.6 9.6zm3.7 0l16.6-9.6-6.7-3.9c.7-1 1.3-2 1.7-3.2l6.4 3.7v-18l-18.1 10.5v20.5zM96.9 57.6l18.6 10.7L134 57.6 117.3 48v7.6c-.6-.1-1.2-.1-1.8-.1-.6 0-1.2 0-1.8.1V48l-16.8 9.6zm20.2-13.9l20.3 11.7c1 .6 1.6 1.7 1.6 2.8v23.5c0 1.2-.6 2.2-1.6 2.8l-20.3 11.7c-1 .6-2.3.6-3.3 0L93.5 84.5c-1-.6-1.6-1.7-1.6-2.8V58.2c0-1.2.6-2.2 1.6-2.8l20.3-11.7c1-.6 2.3-.6 3.3 0z" class="st3"/> </g> <path id="s" d="M74.6 113c-1.8-1-3.5-1.5-5.2-1.5-1.4 0-2.3.6-2.3 1.5 0 2.7 10.1.4 10.1 7.7 0 3.3-2.9 6-7.6 6-2.1 0-4.7-.5-6.4-1.4l-.1-.1c-.3-.2-.3-.5-.2-.8l1.2-2.7c.1-.3.5-.5.9-.3.1 0 .1.1.2.1 1.5.6 3.1 1 4.6 1 2.2 0 2.9-.6 2.9-1.7 0-3-10.1-.8-10.1-7.7 0-3.1 2.7-5.8 7-5.8 2.1 0 5 .7 6.9 1.8.1 0 .1.1.2.1.3.2.4.5.3.8l-1.2 2.7c-.1.3-.5.5-.9.3h-.3z" class="st4"/> <path id="q" d="M93.6 107.8h3.2c.4 0 .7.3.7.7v25.9c0 .4-.3.7-.7.7h-3.2c-.4 0-.7-.3-.7-.7v-9.1c-1.2.8-2.9 1.4-4.7 1.4-5.4 0-9.6-4.3-9.6-9.7 0-5.4 4.1-9.7 9.6-9.7 1.8 0 3.5.6 4.7 1.4v-.1c0-.5.3-.8.7-.8zm-.7 12.4v-6.5c-1.3-1.3-2.8-2.1-4.5-2.1-2.9 0-5.1 2.3-5.1 5.4s2.2 5.4 5.1 5.4c1.7-.1 3.2-.7 4.5-2.2z" class="st4"/> <path id="r" d="M112.5 107.8c-1-.4-2-.6-3-.6-1.8 0-3.5.6-4.9 1.4v-.2c0-.3-.2-.5-.5-.5h-3.4c-.3 0-.5.2-.5.5v17.8c0 .3.2.5.5.5h3.4c.3 0 .5-.2.5-.5v-12.6c1.1-1.2 2.8-1.9 4.6-1.9.4 0 .9 0 1.5.2.3.1.6-.1.7-.4l1.3-2.9c.1-.4 0-.7-.2-.8z" class="st4"/> <path id="e" d="M129 124.7c-1.7 1-4.2 2-6.7 2-6 0-10.3-4.4-10.3-9.9 0-5.3 3.7-9.6 9.4-9.6 5.2 0 8.4 4.4 8.4 9 0 .4 0 .9-.1 1.2 0 .3-.3.6-.7.6h-12.5c.5 2.8 2.8 4.5 5.8 4.5 1.7 0 3.4-.5 5.1-1.4.3-.2.6-.1.8.2l1.2 2.6c.1.2 0 .4-.2.6-.2.1-.2.2-.2.2zm-12.4-10h8.5c-.2-1.8-1.9-3.3-3.9-3.3-2.5-.1-4 1.4-4.6 3.3z" class="st4"/> <path id="e_1_" d="M148.7 124.7c-1.7 1-4.2 2-6.7 2-6 0-10.3-4.4-10.3-9.9 0-5.3 3.7-9.6 9.4-9.6 5.2 0 8.4 4.4 8.4 9 0 .4 0 .9-.1 1.2 0 .3-.3.6-.7.6h-12.5c.5 2.8 2.8 4.5 5.8 4.5 1.7 0 3.4-.5 5.1-1.4.3-.2.6-.1.8.2l1.2 2.6c.1.2 0 .4-.2.6-.2.1-.2.2-.2.2zm-12.4-10h8.5c-.2-1.8-1.9-3.3-3.9-3.3-2.5-.1-4 1.4-4.6 3.3z" class="st4"/> <path id="n" d="M151.5 108.5V126c0 .4.3.7.7.7h3.2c.4 0 .7-.3.7-.7v-12.5c1.1-1.2 2.8-1.9 4.6-1.9 2.9 0 4.5 1.6 4.5 4.7v9.7c0 .4.3.7.7.7h3.2c.4 0 .7-.3.7-.7v-10.2c0-5.2-2.9-8.5-8.8-8.5-1.8 0-3.5.6-4.9 1.4v-.1c0-.4-.3-.7-.7-.7h-3.2c-.4-.1-.7.2-.7.6z" class="st4"/> </svg> <h1>Uh Oh! Sqreen has detected an attack.</h1> <p>If you are the application owner, check the Sqreen <a href="https://my.sqreen.com/">dashboard</a> for more information.</p></div></body></html>`

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
	_, _ = w.Write([]byte(sqreenBlockPage))
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

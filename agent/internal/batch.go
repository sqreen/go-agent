// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

type Event interface{}

type ExceptionEvent struct {
	err         error
	rulespackID string
}

func NewExceptionEvent(err error, rulespackID string) *ExceptionEvent {
	return &ExceptionEvent{err: err, rulespackID: rulespackID}
}

func (e *ExceptionEvent) GetTime() time.Time {
	if t, ok := e.err.(sqerrors.Timestamper); ok {
		return t.Timestamp()
	}
	return time.Now()
}

func (e *ExceptionEvent) GetKlass() string {
	// By convention, the error identifier is not the error type but rather the
	// first part of the string representation, up to `:`,
	// such as `my error: details`
	str := e.err.Error()
	if i := strings.IndexByte(str, ':'); i != -1 {
		return str[:i]
	}
	// Fallback to the type name
	return fmt.Sprintf("%T", e.err)
}

func (e *ExceptionEvent) GetMessage() string {
	return fmt.Sprintf("%v", e.err)
}

func (e *ExceptionEvent) GetRulespackID() string {
	return e.rulespackID
}

func (e *ExceptionEvent) GetContext() api.ExceptionContext {
	return *api.NewExceptionContextFromFace(e)
}

func (e *ExceptionEvent) GetBacktrace() []api.StackFrame {
	st := sqerrors.StackTrace(e.err)
	if len(st) == 0 {
		return nil
	}
	bt := make([]api.StackFrame, 0, len(st))
	for _, f := range st {
		bt = append(bt, *api.NewStackFrameFromFace(apiStackFrame(f)))
	}
	return bt
}

func (e *ExceptionEvent) GetInfos() interface{} {
	return sqerrors.Info(e.err)
}

type apiStackFrame sqerrors.Frame

func (f apiStackFrame) GetMethod() string {
	return sqerrors.Frame(f).Name()
}

func (f apiStackFrame) GetFile() string {
	return sqerrors.Frame(f).File()
}

func (f apiStackFrame) GetLineNumber() uint32 {
	return uint32(sqerrors.Frame(f).Line())
}

type HTTPRequestRecordEvent struct {
	logger      plog.ErrorLogger
	cfg         *config.Config
	rr          record.RequestRecordForAgentFace
	rulespackID string
}

func NewHTTPRequestRecordEvent(rr record.RequestRecordForAgentFace, rulespackID string, cfg *config.Config, logger plog.ErrorLogger) *HTTPRequestRecordEvent {
	return &HTTPRequestRecordEvent{
		cfg:         cfg,
		logger:      logger,
		rr:          rr,
		rulespackID: rulespackID,
	}
}

func (r *HTTPRequestRecordEvent) GetVersion() string {
	return api.RequestRecordVersion
}

func (r *HTTPRequestRecordEvent) GetRulespackId() string {
	return r.rulespackID
}

func (r *HTTPRequestRecordEvent) GetClientIp() string {
	return r.rr.ClientIP().String()
}

func (e *HTTPRequestRecordEvent) GetRequest() api.RequestRecord_Request {
	return *api.NewRequestRecord_RequestFromFace(&RequestAPIAdaptor{HTTPRequestRecordEvent: e})
}

func (r *HTTPRequestRecordEvent) GetResponse() api.RequestRecord_Response {
	return api.RequestRecord_Response{}
}

func (r *HTTPRequestRecordEvent) GetObserved() api.RequestRecord_Observed {
	events := make([]*api.RequestRecord_Observed_SDKEvent, 0, len(r.rr.Events()))
	for _, event := range r.rr.Events() {
		events = append(events, api.NewRequestRecord_Observed_SDKEventFromFace(event))
	}

	attacks := make([]*api.RequestRecord_Observed_Attack, 0, len(r.rr.Attacks()))
	for _, event := range r.rr.Attacks() {
		attacks = append(attacks, api.NewRequestRecord_Observed_AttackFromFace((*AttackEventAPIAdaptor)(event)))
	}

	return api.RequestRecord_Observed{
		Sdk:     events,
		Attacks: attacks,
	}
}

type AttackEventAPIAdaptor record.AttackEvent

func (a *AttackEventAPIAdaptor) GetRuleName() string {
	return a.Rule
}

func (a *AttackEventAPIAdaptor) GetTest() bool {
	return a.Test
}

func (a *AttackEventAPIAdaptor) GetInfo() interface{} {
	return a.Info
}

func (a *AttackEventAPIAdaptor) GetTime() time.Time {
	return a.Timestamp
}

func (a *AttackEventAPIAdaptor) GetBlock() bool {
	return a.Blocked
}

type RequestAPIAdaptor struct {
	*HTTPRequestRecordEvent
	cache struct {
		remoteIP, remotePort, hostPort string
	}
}

func (a *RequestAPIAdaptor) request() *http.Request {
	return a.rr.Request()
}

func (a *RequestAPIAdaptor) GetRid() string {
	return a.request().Header.Get("X-Request-Id")
}

func (a *RequestAPIAdaptor) GetHeaders() []api.RequestRecord_Request_Header {
	req := a.request()
	trackedHeaders := config.TrackedHTTPHeaders
	if extraHeader := a.cfg.HTTPClientIPHeader(); extraHeader != "" {
		trackedHeaders = append(trackedHeaders, extraHeader)
	}
	headers := make([]api.RequestRecord_Request_Header, 0, len(req.Header))
	for _, header := range trackedHeaders {
		if value := req.Header.Get(header); value != "" {
			headers = append(headers, api.RequestRecord_Request_Header{
				Key:   header,
				Value: value,
			})
		}
	}
	return headers
}

func (a *RequestAPIAdaptor) GetVerb() string {
	return a.request().Method
}

func (a *RequestAPIAdaptor) GetPath() string {
	return a.request().URL.Path
}

func (a *RequestAPIAdaptor) GetRawPath() string {
	return a.request().RequestURI
}

func (a *RequestAPIAdaptor) GetHost() string {
	return a.request().Host
}

func (a *RequestAPIAdaptor) GetPort() string {
	if a.cache.hostPort == "" {
		_, a.cache.hostPort = record.SplitHostPort(a.request().Host)
	}
	return a.cache.hostPort
}

func (a *RequestAPIAdaptor) GetRemoteIp() string {
	if a.cache.remoteIP == "" {
		a.cache.remoteIP, a.cache.remotePort = record.SplitHostPort(a.request().RemoteAddr)
	}
	return a.cache.remoteIP
}

func (a *RequestAPIAdaptor) GetRemotePort() string {
	if a.cache.remotePort != "" {
		a.cache.remoteIP, a.cache.remotePort = record.SplitHostPort(a.request().RemoteAddr)
	}
	return a.cache.remotePort
}

func (a *RequestAPIAdaptor) GetScheme() string {
	if a.request().TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

func (a *RequestAPIAdaptor) GetUserAgent() string {
	return a.request().UserAgent()
}

func (a *RequestAPIAdaptor) GetReferer() string {
	if a.cfg.StripHTTPReferer() {
		return ""
	}
	return a.request().Referer()
}

func (a *RequestAPIAdaptor) GetParameters() api.RequestRecord_Request_Parameters {
	req := a.request()
	// .Form and .PostForm are taken as is, without calling `ParseForm()` so
	// that we take what has been done during the request handling.
	// So they can be nil even if there were form parameters in the
	// body.
	return api.RequestRecord_Request_Parameters{
		Query: req.Form,
		Form:  req.PostForm,
	}
}

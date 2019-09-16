// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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

func (r *HTTPRequestRecordEvent) GetRequest() api.RequestRecord_Request {
	req := r.rr.Request()

	trackedHeaders := config.TrackedHTTPHeaders
	if extraHeader := r.cfg.HTTPClientIPHeader(); extraHeader != "" {
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

	remoteIP, remotePort := record.SplitHostPort(req.RemoteAddr)
	_, hostPort := record.SplitHostPort(req.Host)

	var scheme string
	if req.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}

	requestId := req.Header.Get("X-Request-Id")
	if requestId == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			// Log the error and continue.
			r.logger.Error(errors.Wrap(err, "could not generate a request id "))
			requestId = ""
		}
		requestId = hex.EncodeToString(uuid[:])
	}

	var referer string
	if !r.cfg.StripHTTPReferer() {
		referer = req.Referer()
	}

	// FIXME: create it from an interface for compile-time error-checking.
	return api.RequestRecord_Request{
		Rid:        requestId,
		Headers:    headers,
		Verb:       req.Method,
		RawPath:    req.RequestURI,
		Path:       req.URL.Path,
		Host:       req.Host,
		Port:       hostPort,
		RemoteIp:   remoteIP,
		RemotePort: remotePort,
		Scheme:     scheme,
		UserAgent:  req.UserAgent(),
		Referer:    referer,
	}
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

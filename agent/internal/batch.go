// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
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
	rr          *HTTPRequestRecord
	rulespackID string
}

func NewHTTPRequestRecordEvent(rr *HTTPRequestRecord, rulespackID string) *HTTPRequestRecordEvent {
	return &HTTPRequestRecordEvent{
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
	return getClientIP(r.rr.request, r.rr.agent.config).String()
}

func (r *HTTPRequestRecordEvent) GetRequest() api.RequestRecord_Request {
	req := r.rr.request

	trackedHeaders := config.TrackedHTTPHeaders
	if extraHeader := r.rr.agent.config.HTTPClientIPHeader(); extraHeader != "" {
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

	remoteIP, remotePort := splitHostPort(req.RemoteAddr)
	_, hostPort := splitHostPort(req.Host)

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
			r.rr.agent.logger.Error(errors.Wrap(err, "could not generate a request id "))
			requestId = ""
		}
		requestId = hex.EncodeToString(uuid[:])
	}

	var referer string
	if !r.rr.agent.config.StripHTTPReferer() {
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
	events := make([]*api.RequestRecord_Observed_SDKEvent, 0, len(r.rr.events))
	for _, event := range r.rr.events {
		events = append(events, api.NewRequestRecord_Observed_SDKEventFromFace(event))
	}

	return api.RequestRecord_Observed{
		Sdk: events,
	}
}

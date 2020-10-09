// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
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
	if t, ok := sqerrors.Timestamp(e.err); ok {
		return t
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
	return errorStackTraceAPIAdapter(st).GetBacktrace()
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

func (f apiStackFrame) GetLineNumber() int {
	return sqerrors.Frame(f).Line()
}

type closedHTTPRequestContextEvent struct {
	start, finish time.Time
	rulepackID    string
	request       types.RequestReader
	response      types.ResponseFace
	events        event.Recorded
}

func (e *closedHTTPRequestContextEvent) shouldSend() bool {
	// Keeping this function simple to read instead of combining every condition
	// [and mess up]
	if len(e.events.AttackEvents) > 0 {
		return true
	}

	onlyIdentifies := true
	for _, e := range e.events.CustomEvents {
		if e.Method != event.SDKMethodIdentify {
			onlyIdentifies = false
			break
		}
	}
	if !onlyIdentifies {
		return true
	}

	return false
}

func newClosedHTTPRequestContextEvent(rulepackID string, start, finish time.Time, response types.ResponseFace, request types.RequestReader, events event.Recorded) *closedHTTPRequestContextEvent {
	return &closedHTTPRequestContextEvent{
		start:      start,
		finish:     finish,
		rulepackID: rulepackID,
		request:    request,
		response:   response,
		events:     events,
	}
}

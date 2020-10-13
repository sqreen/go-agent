// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import (
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

type RootProtectionContext interface {
	SqreenTime() *sqtime.SharedStopWatch
	DeadlineExceeded(needed time.Duration) (exceeded bool)
	FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error)
	FindActionByUserID(userID map[string]string) (action actor.Action, exists bool)
	IsIPAllowed(ip net.IP) bool
	IsPathAllowed(path string) bool
	Config() ConfigReader
	Close(ClosedProtectionContextFace)
}

type ConfigReader interface {
	HTTPClientIPHeader() string
	HTTPClientIPHeaderFormat() string
}

// RequestReader is the read-only interface to the request.
type RequestReader interface {
	Header(header string) (value *string)
	Headers() http.Header
	Method() string
	URL() *url.URL
	RequestURI() string
	Host() string
	RemoteAddr() string
	IsTLS() bool
	UserAgent() string
	Referer() string
	QueryForm() url.Values
	PostForm() url.Values
	ClientIP() net.IP
	// Params returns the request parameters parsed by the handler so far at the
	// moment of the call.
	Params() RequestParamMap
	// Body returns the body bytes read by the handler so far at the moment of the
	// call.
	Body() []byte
}

type (
	// RequestParamValueMap is the map of request param values per param name.
	// The slice of values allows to have multiple values per param name. For
	// example, the same request parameter name can be use both in the query and
	// form parameters.
	RequestParamMap map[string]RequestParamValueSlice
	// RequestParamValueSlice is the slice of request param values.
	// Note that this is a type alias to allow conversions to []interface{},
	// so that map[string]RequestParamValueSlice and map[string][]interface{} are
	// considered the same type.
	RequestParamValueSlice = []interface{}
)

func (m *RequestParamMap) Add(key string, value interface{}) {
	if *m == nil {
		*m = make(RequestParamMap)
	}
	params := (*m)[key]
	(*m)[key] = append(params, value)
}

// ResponseWriter is the response writer interface.
type ResponseWriter interface {
	http.ResponseWriter
	io.StringWriter
}

// ResponseFace is the interface to the response that was sent by the handler.
type ResponseFace interface {
	Status() int
	ContentType() string
	ContentLength() int64
}

type ClosedProtectionContextFace interface {
	Response() ResponseFace
	Request() RequestReader
	Events() event.Recorded
	Start() time.Time
	Duration() time.Duration
	SqreenTime() time.Duration
}

type WriteAfterCloseError struct{}

func (WriteAfterCloseError) Error() string { return "response write after close" }

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

type RootProtectionContext interface {
	Context() context.Context
	CancelContext()
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

type (
	// RequestReader is the read-only interface to the request.
	RequestReader interface {
		Headers() http.Header
		Method() string
		URL() *url.URL
		Host() string
		RemoteAddr() string
		UserAgent() string
		Referer() string
		Transport() string
		RequestURI() string
	}

	// RequestPathParamsGetter is an optional interface for HTTP frameworks
	// having request path parameters such as `/path/:arg1/:arg2`.
	RequestPathParamsGetter interface {
		PathParams() interface{}
	}

	RequestBindingAccessorReader interface {
		RequestReader
		Body() []byte
		Params() RequestParamMap
	}

	ClosedRequestReader interface {
		ClientIP() net.IP
		Params() RequestParamMap
		RequestReader
	}
)

type (
	// RequestParamMap is the map of request param values per span address name.
	RequestParamMap map[string]interface{}
)

// ResponseWriter is the response writer interface.
type ResponseWriter interface {
	http.ResponseWriter
}

// ResponseFace is the interface to the response that was sent by the handler.
type ResponseFace interface {
	Status() int
	ContentType() string
	ContentLength() int64
}

type ClosedProtectionContextFace interface {
	Response() ResponseFace
	Request() ClosedRequestReader
	Events() event.Recorded
	Start() time.Time
	Duration() time.Duration
	SqreenTime() time.Duration
}

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/sqreen/go-agent/internal/event"
)

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
	Form() url.Values
	PostForm() url.Values
	ClientIP() net.IP
	FrameworkParams() url.Values
}

// ResponseWriter is the response writer interface.
type ResponseWriter interface {
	http.ResponseWriter
	io.StringWriter
	// Close the response writer and return the written response.
	// Any writer methods following this call should be ignored.
	//closeResponseWriter() ResponseFace
}

// ResponseFace is the interface to the response that was sent by the handler.
type ResponseFace interface {
	Status() int
	ContentType() string
	ContentLength() int64
}

type ClosedRequestContextFace interface {
	Response() ResponseFace
	Request() RequestReader
	Events() event.Recorded
}

type WriteAfterCloseError struct{}

func (WriteAfterCloseError) Error() string { return "response write after close" }

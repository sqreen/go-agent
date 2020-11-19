// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package mockups

import (
	"net"
	"net/http"
	"net/url"

	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/stretchr/testify/mock"
)

type ResponseWriterMockup struct {
	mock.Mock
}

func (r *ResponseWriterMockup) Header() http.Header {
	h, _ := r.Called().Get(0).(http.Header)
	return h
}

func (r *ResponseWriterMockup) ExpectHeader() *mock.Call {
	return r.On("Header")
}

func (r *ResponseWriterMockup) Write(bytes []byte) (int, error) {
	ret := r.Called(bytes)
	return ret.Int(0), ret.Error(1)
}

func (r *ResponseWriterMockup) WriteHeader(statusCode int) {
	r.Called(statusCode)
}

func (r *ResponseWriterMockup) ExpectWriteHeader(statusCode int) *mock.Call {
	return r.On("WriteHeader", statusCode)
}

func (r *ResponseWriterMockup) WriteString(s string) (n int, err error) {
	ret := r.Called(s)
	return ret.Int(0), ret.Error(1)
}

type RequestReaderMockup struct {
	mock.Mock
}

func (r *RequestReaderMockup) Body() []byte {
	value, _ := r.Called().Get(0).([]byte)
	return value
}

func (r *RequestReaderMockup) Header(header string) (value *string) {
	value, _ = r.Called(header).Get(0).(*string)
	return value
}

func (r *RequestReaderMockup) Headers() http.Header {
	h, _ := r.Called().Get(0).(http.Header)
	return h
}

func (r *RequestReaderMockup) ExpectHeaders() *mock.Call {
	return r.On("Headers")
}

func (r *RequestReaderMockup) Method() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) URL() *url.URL {
	u, _ := r.Called().Get(0).(*url.URL)
	return u
}

func (r *RequestReaderMockup) ExpectURL() *mock.Call {
	return r.On("URL")
}

func (r *RequestReaderMockup) RequestURI() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) Host() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) RemoteAddr() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) ExpectRemoteAddr() *mock.Call {
	return r.On("RemoteAddr")
}

func (r *RequestReaderMockup) IsTLS() bool {
	return r.Called().Bool(0)
}

func (r *RequestReaderMockup) UserAgent() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) Referer() string {
	return r.Called().String(0)
}

func (r *RequestReaderMockup) QueryForm() url.Values {
	v, _ := r.Called().Get(0).(url.Values)
	return v
}

func (r *RequestReaderMockup) PostForm() url.Values {
	v, _ := r.Called().Get(0).(url.Values)
	return v
}

func (r *RequestReaderMockup) ClientIP() net.IP {
	ip, _ := r.Called().Get(0).(net.IP)
	return ip
}

func (r *RequestReaderMockup) ExpectClientIP() *mock.Call {
	return r.On("ClientIP")
}

func (r *RequestReaderMockup) Params() types.RequestParamMap {
	v, _ := r.Called().Get(0).(types.RequestParamMap)
	return v
}

func (r *RequestReaderMockup) ExpectMethod() *mock.Call {
	return r.On("Method")
}

func (r *RequestReaderMockup) ExpectRequestURI() *mock.Call {
	return r.On("RequestURI")
}

func (r *RequestReaderMockup) ExpectHost() *mock.Call {
	return r.On("Host")
}

func (r *RequestReaderMockup) ExpectIsTLS() *mock.Call {
	return r.On("IsTLS")
}

func (r *RequestReaderMockup) ExpectUserAgent() *mock.Call {
	return r.On("UserAgent")
}

func (r *RequestReaderMockup) ExpectReferer() *mock.Call {
	return r.On("Referer")
}

func (r *RequestReaderMockup) ExpectQueryForm() *mock.Call {
	return r.On("QueryForm")
}

func (r *RequestReaderMockup) ExpectPostForm() *mock.Call {
	return r.On("PostForm")
}

func (r *RequestReaderMockup) ExpectParams() *mock.Call {
	return r.On("Params")
}

func (r *RequestReaderMockup) ExpectBody() *mock.Call {
	return r.On("Body")
}

type ResponseMockup struct {
	mock.Mock
}

func (r *ResponseMockup) Status() int {
	return r.Called().Int(0)
}

func (r *ResponseMockup) ExpectStatus() *mock.Call {
	return r.On("Status")
}

func (r *ResponseMockup) ContentType() string {
	return r.Called().String(0)
}

func (r *ResponseMockup) ExpectContentType() *mock.Call {
	return r.On("ContentType")
}

func (r *ResponseMockup) ContentLength() int64 {
	return r.Called().Get(0).(int64)
}

func (r *ResponseMockup) ExpectContentLength() *mock.Call {
	return r.On("ContentLength")
}

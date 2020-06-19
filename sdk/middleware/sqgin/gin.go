// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package sqgin

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	gingonic "github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/internal"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
)

// Middleware is Sqreen's middleware function for Gin to monitor and protect the
// requests Gin receives.
//
// SDK methods can be called from request handlers by using the request context.
// It can be retrieved from the request context using `sdk.FromContext()` or
// on a Gin's context.
//
// Usage example:
//
//	router := gin.Default()
//	router.Use(sqgin.Middleware())
//
//	router.GET("/", func(c *gin.Context) {
//		// Accessing the SDK through Gin's context
//		sdk.FromContext(c).TrackEvent("my.event.one")
//		foo(c.Request)
//	}
//
//	func foo(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//		// ...
//	}
//
//	router.GET("/", func(c *gin.Context) {
//		// Example of globally identifying a user and checking if the request
//		// should be aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sdk.FromContext(c).ForUser(uid)
//		// Globally associate this user to the current request and check if it got
//		// blocked.
//		if err := sqUser.Identify(); err != nil {
//			// Return to stop further handling the request
//			return
//		}
//		// ... not blocked ...
//	}
//
func Middleware() gingonic.HandlerFunc {
	internal.Start()
	return func(c *gingonic.Context) {
		middlewareHandler(internal.Agent(), c)
	}
}

func middlewareHandler(agent protection_context.AgentFace, c *gingonic.Context) {
	if agent == nil {
		// The agent is disabled or not yet started.
		c.Next()
		return
	}

	requestReader := &requestReaderImpl{c: c}
	responseWriter := &responseWriterImpl{c: c}

	reqCtx, cancelHandlerContext := context.WithCancel(c.Request.Context())
	defer cancelHandlerContext()

	ctx := http_protection.NewRequestContext(agent, responseWriter, requestReader, cancelHandlerContext)
	if ctx == nil {
		c.Next()
		return
	}

	defer func() {
		_ = ctx.Close(responseWriter.closeResponseWriter())
	}()

	c.Set(protection_context.ContextKey.String, ctx)
	c.Request = c.Request.WithContext(context.WithValue(reqCtx, protection_context.ContextKey, ctx))

	if err := ctx.Before(); err != nil {
		c.Abort()
		return
	}
	c.Next()
	// Handler-based protection such as user security responses or RASP
	// protection may lead to aborted requests.
	if c.IsAborted() {
		return
	}
	if err := ctx.After(); err != nil {
		c.Abort()
		return
	}
}

type requestReaderImpl struct {
	c *gingonic.Context
}

func (r *requestReaderImpl) UserAgent() string {
	return r.c.Request.UserAgent()
}

func (r *requestReaderImpl) Referer() string {
	return r.c.Request.Referer()
}

func (r *requestReaderImpl) ClientIP() net.IP {
	return nil // Delegated to the middleware according the agent configuration
}

func (r *requestReaderImpl) Method() string {
	return r.c.Request.Method
}

func (r *requestReaderImpl) URL() *url.URL {
	return r.c.Request.URL
}

func (r *requestReaderImpl) RequestURI() string {
	return r.c.Request.RequestURI
}

func (r *requestReaderImpl) Host() string {
	return r.c.Request.Host
}

func (r *requestReaderImpl) IsTLS() bool {
	return r.c.Request.TLS != nil
}

func (r *requestReaderImpl) FrameworkParams() url.Values {
	params := r.c.Params
	res := url.Values{}
	for _, param := range params {
		res[param.Key] = []string{param.Value}
	}
	return res
}

func (r *requestReaderImpl) Form() url.Values {
	_ = r.c.Request.ParseForm()
	return r.c.Request.Form
}

func (r *requestReaderImpl) PostForm() url.Values {
	_ = r.c.Request.ParseForm()
	return r.c.Request.PostForm
}

func (r *requestReaderImpl) Headers() http.Header {
	return r.c.Request.Header
}

func (r *requestReaderImpl) Header(h string) *string {
	headers := r.c.Request.Header
	if headers == nil {
		return nil
	}
	v := headers[textproto.CanonicalMIMEHeaderKey(h)]
	if len(v) == 0 {
		return nil
	}
	return &v[0]
}

func (r *requestReaderImpl) RemoteAddr() string {
	return r.c.Request.RemoteAddr
}

type responseWriterImpl struct {
	c      *gingonic.Context
	closed bool
	// Gin allows overwriting the status field even when it was already done and
	// sent over the network. It can therefore lead to a status code distinct from
	// what was actually sent. To avoid this problem, we record the status code
	// we see going through this wrapper. Note that the absence of dynamic
	// dispatch in Go can allow to avoid this wrapper.
	writtenStatus int
}

func (w *responseWriterImpl) closeResponseWriter() types.ResponseFace {
	if !w.closed {
		w.closed = true
	}
	return newObservedResponse(w)
}

func (w *responseWriterImpl) Header() http.Header {
	return w.c.Writer.Header()
}

func (w *responseWriterImpl) Write(b []byte) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	return w.c.Writer.Write(b)
}

func (w *responseWriterImpl) WriteString(s string) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	return io.WriteString(w.c.Writer, s)
}

// Static assert that the io.StringWriter is implemented
var _ io.StringWriter = (*responseWriterImpl)(nil)

func (w *responseWriterImpl) WriteHeader(statusCode int) {
	if w.closed {
		return
	}
	w.c.Writer.WriteHeader(statusCode)
	w.writtenStatus = statusCode
}

// response observed by the response writer
type observedResponse struct {
	contentType   string
	contentLength int64
	status        int
}

func newObservedResponse(r *responseWriterImpl) *observedResponse {
	// Content-Type will be not empty only when explicitly set.
	// It could be guessed as net/http does. Not implemented for now.
	ct := r.Header().Get("Content-Type")

	// Content-Length is either explicitly set or the amount of written data.
	cl := int64(r.c.Writer.Size())
	if contentLength := r.Header().Get("Content-Length"); contentLength != "" {
		if l, err := strconv.ParseInt(contentLength, 10, 0); err == nil {
			cl = l
		}
	}

	// Take the status code we observed, and Gin's if none.
	status := r.writtenStatus
	if status == 0 {
		status = r.c.Writer.Status()
	}

	return &observedResponse{
		contentType:   ct,
		contentLength: cl,
		status:        status,
	}
}

func (r *observedResponse) Status() int {
	if status := r.status; status != 0 {
		return status
	}
	// Default net/http status is 200
	return http.StatusOK
}

func (r *observedResponse) ContentType() string {
	return r.contentType
}

func (r *observedResponse) ContentLength() int64 {
	return r.contentLength
}

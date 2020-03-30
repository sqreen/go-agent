// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"github.com/sqreen/go-agent/internal"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
)

// Middleware is Sqreen's middleware function for `net/http` to monitor and
// protect received requests.
//
// SDK methods can be called from request handlers by using the request context.
// It can be retrieved from the request context using `sdk.FromContext()`.
//
// Usage example:
//
//	fn := func(w http.ResponseWriter, r *http.Request) {
//		// Get the requestImplType record.
//		sqreen := sdk.FromContext(r.Context())
//
//		// Example of sending a custom event.
//		sqreen.TrackEvent("my.event")
//
//		// Example of globally identifying a user and checking if the request
//		// should be aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sqreen.ForUser(uid)
//		// Globally associate this user to the current request and check if it got
//		// blocked.
//		if err := sqUser.Identify(); err != nil {
//			// Return to stop further handling the request
//			return
//		}
//		// User not blocked
//		fmt.Fprintf(w, "OK")
//	}
//	http.Handle("/foo", sqhttp.Middleware(http.HandlerFunc(fn)))
//
func Middleware(next http.Handler) http.Handler {
	internal.Start()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middlewareHandler(internal.Agent(), next, w, r)
	})
}
func middlewareHandler(agent protection_context.AgentFace, next http.Handler, w http.ResponseWriter, r *http.Request) {
	if agent == nil {
		next.ServeHTTP(w, r)
		return
	}

	// requestReader is a pointer value in order to change the inner request
	// pointer with the new one created by http.(*Request).WithContext below
	requestReader := &requestReaderImpl{Request: r}
	responseWriter := &responseWriterImpl{ResponseWriter: w}

	reqCtx, cancelHandlerContext := context.WithCancel(r.Context())
	defer cancelHandlerContext()

	ctx := http_protection.NewRequestContext(agent, responseWriter, requestReader, cancelHandlerContext)
	if ctx == nil {
		next.ServeHTTP(w, r)
		return
	}
	defer func() {
		ctx.Close(responseWriter.closeResponseWriter())
	}()

	reqCtx = context.WithValue(reqCtx, protection_context.ContextKey, ctx)
	requestReader.Request = r.WithContext(reqCtx)

	if err := ctx.Before(); err != nil {
		return
	}
	next.ServeHTTP(responseWriter, requestReader.Request)
	if err := ctx.After(); err != nil {
		return
	}
}

type requestReaderImpl struct {
	*http.Request
}

func (r *requestReaderImpl) Header(h string) (value *string) {
	headers := r.Request.Header
	if headers == nil {
		return nil
	}
	v := headers[textproto.CanonicalMIMEHeaderKey(h)]
	if len(v) == 0 {
		return nil
	}
	return &v[0]
}

func (r *requestReaderImpl) FrameworkParams() url.Values {
	return nil // none using net/http
}

func (r *requestReaderImpl) ClientIP() net.IP {
	return nil // Delegated to the middleware according the agent configuration
}

func (r *requestReaderImpl) Method() string {
	return r.Request.Method
}

func (r *requestReaderImpl) URL() *url.URL {
	return r.Request.URL
}

func (r *requestReaderImpl) RequestURI() string {
	return r.Request.RequestURI
}

func (r *requestReaderImpl) Host() string {
	return r.Request.Host
}

func (r *requestReaderImpl) IsTLS() bool {
	return r.Request.TLS != nil
}

func (r *requestReaderImpl) Form() url.Values {
	_ = r.Request.ParseForm()
	return r.Request.Form
}

func (r *requestReaderImpl) PostForm() url.Values {
	_ = r.Request.ParseForm()
	return r.Request.PostForm
}

func (r *requestReaderImpl) Headers() http.Header {
	return r.Request.Header
}

func (r *requestReaderImpl) RemoteAddr() string {
	return r.Request.RemoteAddr
}

type responseWriterImpl struct {
	http.ResponseWriter
	status  int
	written int
	closed  bool
}

// response observed by the response writer
type observedResponse struct {
	contentType   string
	contentLength int
	status        int
}

func newObservedResponse(r *responseWriterImpl) *observedResponse {
	// Content-Type will be not empty only when explicitly set.
	// It could be guessed as net/http does. Not implemented for now.
	ct := r.Header().Get("Content-Type")
	// Content-Length is either explicitly set or the amount of written data.
	cl := r.written
	if contentLength := r.Header().Get("Content-Length"); contentLength != "" {
		if l, err := strconv.ParseInt(contentLength, 10, 0); err == nil {
			cl = int(l)
		}
	}

	return &observedResponse{
		contentType:   ct,
		contentLength: cl,
		status:        r.status,
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
	return int64(r.contentLength)
}

func (w *responseWriterImpl) closeResponseWriter() types.ResponseFace {
	if !w.closed {
		w.closed = true
	}
	return newObservedResponse(w)
}

func (w *responseWriterImpl) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *responseWriterImpl) Write(b []byte) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	written, err := w.ResponseWriter.Write(b)
	if err == nil {
		w.written += written
	}
	return written, err
}

func (w *responseWriterImpl) WriteString(s string) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	written, err := io.WriteString(w.ResponseWriter, s)
	if err == nil {
		w.written += written
	}
	return written, err
}

// Static assert that the io.StringWriter is implemented
var _ io.StringWriter = &responseWriterImpl{}

func (w *responseWriterImpl) WriteHeader(statusCode int) {
	if w.closed {
		return
	}
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

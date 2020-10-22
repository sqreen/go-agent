// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/sqreen/go-agent/internal"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/sdk"
)

// FromContext allows to access the HTTPRequestRecord from Echo request handlers
// if present, and nil otherwise. The value is stored in handler contexts by the
// middleware function, and is of type *HTTPRequestRecord.
//
// Note that Echo's context does not implement the `context.Context` interface,
// so `sdk.FromContext()` cannot be used with it, hence another `FromContext()`
// in this package to access the SDK context value from Echo's context.
// `sdk.FromContext()` can still be used on the request context.
func FromContext(c echo.Context) sdk.Context {
	return sdk.FromContext(c.Request().Context())
}

// Middleware is Sqreen's middleware function for echo to monitor and protect the
// requests echo receives.
//
// SDK methods can be called from request handlers by using the request context.
// It can be retrieved from the request context using `sdk.FromContext()` or
// on a echo's context.
//
// Usage example:
//
//	e := echo.New()
//	e.Use(sqecho.Middleware())
//
//	e.GET("/", func(c echo.Context) error {
//		// Accessing the SDK through Echo's context
//		sqecho.FromContext(c).TrackEvent("my.event.one")
//		foo(c.Request())
//		return nil
//	}
//
//	func foo(req *http.Request) {
//		// Accessing the SDK through the request context
//		sdk.FromContext(req.Context()).TrackEvent("my.event.two")
//	}
//
//	e.GET("/", func(c echo.Context) {
//		// Globally identifying a user and checking if the request should be
//		// aborted.
//		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//		sqUser := sqecho.FromContext(c).ForUser(uid)
//		// Globally associate this user to the current request and check if it got
//		// blocked.
//		if err := sqUser.Identify(); err != nil {
//			// Return to stop further handling the request
//			return err
//		}
//		// ... not blocked ...
//		return nil
//	}
//
func Middleware() echo.MiddlewareFunc {
	internal.Start()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, cancel := internal.NewRootHTTPProtectionContext(c.Request().Context())
			if ctx == nil {
				return next(c)
			}
			defer cancel()
			return middlewareHandlerFromRootProtectionContext(ctx, next, c)
		}
	}
}

func middlewareHandlerFromRootProtectionContext(ctx types.RootProtectionContext, next echo.HandlerFunc, c echo.Context) error {
	w := &responseWriterImpl{c: c}
	r := &requestReaderImpl{c: c}
	p := http_protection.NewProtectionContext(ctx, w, r)
	if p == nil {
		return next(c)
	}

	defer func() {
		p.Close(w.closeResponseWriter())
	}()

	return middlewareHandlerFromProtectionContext(p, next, c)
}

type protectionContext interface {
	WrapRequest(*http.Request) *http.Request
	Before() error
	After() error
}

func middlewareHandlerFromProtectionContext(p protectionContext, next echo.HandlerFunc, c echo.Context) error {
	c.SetRequest(p.WrapRequest(c.Request()))
	c.Set(protectioncontext.ContextKey.String, p)

	if err := p.Before(); err != nil {
		return err
	}
	if err := next(c); err != nil {
		// Handler-based protection such as user security responses or RASP
		// protection may lead to aborted requests bubbling up the error that
		// was returned.
		return err
	}
	return p.After()
}

type requestReaderImpl struct {
	c              echo.Context
	bodyReadBuffer bytes.Buffer
	queryForm      url.Values
}

func (r *requestReaderImpl) Body() []byte {
	return nil
}

func (r *requestReaderImpl) UserAgent() string {
	return r.c.Request().UserAgent()
}

func (r *requestReaderImpl) Referer() string {
	return r.c.Request().Referer()
}

func (r *requestReaderImpl) ClientIP() net.IP {
	return nil // Delegated to the middleware according the rootProtectectionContext configuration
}

func (r *requestReaderImpl) Method() string {
	return r.c.Request().Method
}

func (r *requestReaderImpl) URL() *url.URL {
	return r.c.Request().URL
}

func (r *requestReaderImpl) RequestURI() string {
	return r.c.Request().RequestURI
}

func (r *requestReaderImpl) Host() string {
	return r.c.Request().Host
}

func (r *requestReaderImpl) IsTLS() bool {
	return r.c.IsTLS()
}

func (r *requestReaderImpl) Params() types.RequestParamMap {
	params := r.c.ParamNames()
	l := len(params)
	if l == 0 {
		return nil
	}

	m := make(types.RequestParamMap, l)
	for _, key := range params {
		m.Add(key, r.c.Param(key))
	}
	return m
}

func (r *requestReaderImpl) QueryForm() url.Values {
	if r.queryForm == nil {
		r.queryForm = r.c.Request().URL.Query()
	}
	return r.queryForm
}

func (r *requestReaderImpl) PostForm() url.Values {
	return r.c.Request().PostForm
}

func (r *requestReaderImpl) Headers() http.Header {
	return r.c.Request().Header
}

func (r *requestReaderImpl) Header(h string) *string {
	headers := r.c.Request().Header
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
	return r.c.Request().RemoteAddr
}

type responseWriterImpl struct {
	c      echo.Context
	closed bool
}

func (w *responseWriterImpl) closeResponseWriter() types.ResponseFace {
	if !w.closed {
		w.closed = true
	}
	return newObservedResponse(w)
}

func (w *responseWriterImpl) Header() http.Header {
	return w.c.Response().Header()
}

func (w *responseWriterImpl) Write(b []byte) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	return w.c.Response().Write(b)
}

func (w *responseWriterImpl) WriteString(s string) (int, error) {
	if w.closed {
		return 0, types.WriteAfterCloseError{}
	}
	return io.WriteString(w.c.Response(), s)
}

// Static assert that the io.StringWriter is implemented
var _ io.StringWriter = (*responseWriterImpl)(nil)

func (w *responseWriterImpl) WriteHeader(statusCode int) {
	if w.closed {
		return
	}
	w.c.Response().WriteHeader(statusCode)
}

// response observed by the response writer
type observedResponse struct {
	contentType   string
	contentLength int64
	status        int
}

func newObservedResponse(r *responseWriterImpl) *observedResponse {
	response := r.c.Response()

	headers := response.Header()

	// Content-Type will be not empty only when explicitly set.
	// It could be guessed as net/http does. Not implemented for now.
	ct := headers.Get("Content-Type")

	// Content-Length is either explicitly set or the amount of written data. It's
	// 0 by default with Echo.
	cl := response.Size
	if cl == 0 {
		if contentLength := headers.Get("Content-Length"); contentLength != "" {
			if l, err := strconv.ParseInt(contentLength, 10, 0); err == nil {
				cl = l
			}
		}
	}

	status := response.Status

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

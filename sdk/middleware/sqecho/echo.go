// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqecho

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"github.com/labstack/echo"
	"github.com/sqreen/go-agent/internal"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
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
func FromContext(c echo.Context) *sdk.Context {
	return sdk.FromContext(c.Request().Context())
}

// Middleware is Sqreen's middleware function for Echo to monitor and protect
// the requests Echo receives. In protection mode, it can block and redirect
// requests according to their IP addresses or identified users using
// Identify()` and `MatchSecurityResponse()` methods.
//
// SDK methods can be called from request handlers by using the request event
// record. It can be accessed using `sdk.FromContext()` on a request context or
// this package's `FromContext()` on an Echo request context. The middleware
// function stores it into both of them.
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
//		sqUser.Identify() // Globally associate this user to the current request
//		if match, err := sqUser.MatchSecurityResponse(); match {
//			// Return to stop further handling the request and let Sqreen's
//			// middleware apply and abort the request.
//			return err
//		}
//		// ... not blocked ...
//		return nil
//	}
//
func Middleware() echo.MiddlewareFunc {
	internal.Start()
	return func(handler echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestReader := &requestReaderImpl{c: c}
			responseWriter := &responseWriterImpl{c: c}

			reqCtx, cancelHandlerContext := context.WithCancel(c.Request().Context())
			defer cancelHandlerContext()

			ctx := http_protection.NewRequestContext(internal.Agent(), responseWriter, requestReader, cancelHandlerContext)
			if ctx == nil {
				return handler(c)
			}

			defer func() {
				_ = ctx.Close(responseWriter.closeResponseWriter())
			}()

			c.SetRequest(c.Request().WithContext(context.WithValue(reqCtx, protection_context.ContextKey, ctx)))

			if err := ctx.Before(); err != nil {
				return err
			}
			if err := handler(c); err != nil {
				// Handler-based protection such as user security responses or RASP
				// protection may lead to aborted requests bubbling up the error that
				// was returned.
				return err
			}
			if err := ctx.After(); err != nil {
				return err
			}
			return nil
		}
	}
}

type requestReaderImpl struct {
	c echo.Context
}

func (r *requestReaderImpl) UserAgent() string {
	return r.c.Request().UserAgent()
}

func (r *requestReaderImpl) Referer() string {
	return r.c.Request().Referer()
}

func (r *requestReaderImpl) ClientIP() net.IP {
	return nil // Delegated to the middleware according the agent configuration
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

func (r *requestReaderImpl) FrameworkParams() url.Values {
	res := url.Values{}
	for _, key := range r.c.ParamNames() {
		res[key] = []string{r.c.Param(key)}
	}
	return res
}

func (r *requestReaderImpl) Form() url.Values {
	_ = r.c.Request().ParseForm()
	return r.c.Request().Form
}

func (r *requestReaderImpl) PostForm() url.Values {
	_ = r.c.Request().ParseForm()
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
	status int
	closed bool
}

func (w *responseWriterImpl) closeResponseWriter() types.ResponseFace {
	if !w.closed {
		w.c.Response().Flush()
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
	w.status = statusCode
	w.c.Response().WriteHeader(statusCode)
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
	cl := r.c.Response().Size
	if contentLength := r.Header().Get("Content-Length"); contentLength != "" {
		if l, err := strconv.ParseInt(contentLength, 10, 0); err == nil {
			cl = l
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
	return r.contentLength
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package sqgin

import (
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sqreen/go-agent/internal"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/span"
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
func Middleware() gin.HandlerFunc {
	internal.Start()
	return func(c *gin.Context) {
		ctx, cancel := internal.NewRootHTTPProtectionContext(c.Request.Context())
		if ctx == nil {
			c.Next()
			return
		}
		defer cancel()
		c.Request = c.Request.WithContext(ctx.Context())
		middlewareHandlerFromRootProtectionContext(ctx, c)
	}
}

func middlewareHandlerFromRootProtectionContext(ctx types.RootProtectionContext, c *gin.Context) {
	r := &requestReaderImpl{c: c}
	w := &responseWriter{ResponseWriter: c.Writer}
	p := http_protection.NewProtectionContext(ctx, w, r)
	if p == nil {
		c.Next()
		return
	}

	sp, _ := span.NewSpan(span.WithProtectionContext(p))

	defer func() {
		response := newObservedResponse(w)
		sp.End(response)
		p.Close(response)
	}()

	middlewareHandlerFromProtectionContext(p, c, w)
}

func middlewareHandlerFromProtectionContext(p *http_protection.ProtectionContext, c *gin.Context, w *responseWriter) {
	c.Request = p.WrapRequest(c.Request)
	c.Set(protection_context.ContextKey.String, p)
	c.Writer = w

	if err := p.Before(); err != nil {
		c.Abort()
		return
	}

	sp, err := http_protection.NewHTTPHandlerSpan(p.ClientIP(), p.RequestReader)
	if err != nil {
		return
	}
	defer func() {
		sp.End(newObservedResponse(w))
	}()

	c.Next()
	// Handler-based protection such as user security responses or RASP
	// protection may lead to aborted requests.
	if c.IsAborted() {
		return
	}
	if err := p.After(); err != nil {
		c.Abort()
		return
	}
}

type requestReaderImpl struct {
	c *gin.Context
}

func (r *requestReaderImpl) Transport() string {
	if r.IsTLS() {
		return "https"
	}
	return "http"
}

func (r *requestReaderImpl) PathParams() interface{} {
	return r.pathParams()
}

func (r *requestReaderImpl) pathParams() map[string][]string {
	// TODO: make it once at construction time
	params := r.c.Params
	l := len(params)
	if l == 0 {
		return nil
	}

	m := make(map[string][]string, l)
	for _, param := range params {
		m[param.Key] = append(m[param.Key], param.Value)
	}
	return m
}

func (r *requestReaderImpl) Body() []byte {
	// not called
	// TODO: rework the interfaces to avoid that useless method
	return nil
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

func (r *requestReaderImpl) PostForm() url.Values {
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

type responseWriter struct {
	gin.ResponseWriter
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.status == 0 {
		w.status = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// response observed by the response writer
type observedResponse struct {
	contentType   string
	contentLength int64
	status        int
}

func newObservedResponse(response *responseWriter) *observedResponse {
	headers := response.Header()

	// Content-Type will be not empty only when explicitly set.
	// It could be guessed as net/http does. Not implemented for now.
	ct := headers.Get("Content-Type")

	// Content-Length is either explicitly set or the amount of written data. It's
	// less than 0 when not set by default with Gin.
	cl := int64(response.Size())
	if cl < 0 {
		cl = 0
		if contentLength := headers.Get("Content-Length"); contentLength != "" {
			if l, err := strconv.ParseInt(contentLength, 10, 0); err == nil {
				cl = l
			}
		}
	}

	// Do not use Gin's status code until this gets
	// somehow solved: https://github.com/gin-gonic/gin/pull/2627
	var status int
	if s := response.status; s > 0 {
		status = s
	} else if s := response.ResponseWriter.Status(); s > 0 {
		status = s
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

func (r *observedResponse) Get(key string) (value interface{}, exists bool) {
	switch key {
	default:
		return nil, false
	case "server.response.status":
		return r.status, true
	}
}

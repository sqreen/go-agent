// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqhttp

import (
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"github.com/sqreen/go-agent/internal"
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
		ctx, cancel := internal.NewRootHTTPProtectionContext(r.Context())
		if ctx == nil {
			next.ServeHTTP(w, r)
			return
		}
		defer cancel()
		r = r.WithContext(ctx.Context())
		middlewareHandlerFromRootProtectionContext(ctx, next, w, r)
	})
}

func middlewareHandlerFromRootProtectionContext(ctx types.RootProtectionContext, next http.Handler, w http.ResponseWriter, r *http.Request) {
	// requestReader is a pointer value in order to change the inner request
	// pointer with the new one created by http.(*Request).WithContext below
	requestReader := &requestReaderImpl{Request: r}
	responseWriter, responseWriterObserver := wrapResponseWriter(w)
	p := http_protection.NewProtectionContext(ctx, responseWriter, requestReader)
	if p == nil {
		next.ServeHTTP(w, r)
		return
	}

	defer func() {
		p.Close(newObservedResponse(responseWriterObserver))
	}()

	middlewareHandlerFromProtectionContext(p, next, responseWriter, requestReader)
}

func middlewareHandlerFromProtectionContext(p *http_protection.ProtectionContext, next http.Handler, w http.ResponseWriter, r *requestReaderImpl) {
	r.Request = p.WrapRequest(r.Request)

	// TODO: remove or make it a http span callback
	if err := p.Before(); err != nil {
		return
	}

	sp, err := http_protection.NewHTTPHandlerSpan(nil, p.RequestReader)
	if err != nil {
		return
	}
	defer sp.End(nil)

	next.ServeHTTP(w, r.Request)

	// TODO: remove or make it a http span callback
	if err := p.After(); err != nil {
		return
	}
}

type requestReaderImpl struct {
	*http.Request
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

func (r *requestReaderImpl) pathParams() []string {
	reqURL := r.URL()
	segments := strings.FieldsFunc(reqURL.Path, func(c rune) bool {
		return c == '/'
	})
	if len(segments) == 0 {
		return nil
	}
	return segments
}

func (r *requestReaderImpl) Body() []byte {
	return nil
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

const urlSegmentsFrameworkParamsKey = "URL Segments"

// Params makes its best to return framework parameters, often
// taken from the URL path (eg. a parametrized endpoint `/posts/:id`),
// by returning the list of segments in the URL path. This allows to better cover frameworks using this `net/http`
// middleware, such as Gorilla and Beego.
func (r *requestReaderImpl) Params() types.RequestParamMap {
	segments := r.pathParams()
	if len(segments) == 0 {
		return nil
	}
	return types.RequestParamMap{
		urlSegmentsFrameworkParamsKey: {
			segments,
		},
	}
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

func (r *requestReaderImpl) PostForm() url.Values {
	return r.Request.PostForm
}

func (r *requestReaderImpl) Headers() http.Header {
	return r.Request.Header
}

func (r *requestReaderImpl) RemoteAddr() string {
	return r.Request.RemoteAddr
}

type responseWriterObserver struct {
	http.ResponseWriter
	status  int
	written int
}

// response observed by the response writer
type observedResponse struct {
	contentType   string
	contentLength int
	status        int
}

func newObservedResponse(r *responseWriterObserver) *observedResponse {
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

func (w *responseWriterObserver) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *responseWriterObserver) Write(b []byte) (int, error) {
	written, err := w.ResponseWriter.Write(b)
	if err == nil {
		w.written += written
	}
	return written, err
}

func (w *responseWriterObserver) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

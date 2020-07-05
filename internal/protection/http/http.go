// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/event"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
)

type RequestContext struct {
	*protectioncontext.RequestContext
	RequestReader              types.RequestReader
	ResponseWriter             types.ResponseWriter
	events                     event.Record
	cancelHandlerContextFunc   context.CancelFunc
	contextHandlerCanceledLock sync.RWMutex

	// We are intentionally not using the Context.Err() method here in order to
	// be sure it was canceled by a call to CancelHandlerContext(). Using
	// Context.Err() in order to know this would be also true if for example
	// the parent context timeouts, in which case we mustn't write the blocking
	// response.
	contextHandlerCanceled bool

	requestReader *requestReader
}

// Helper types for callbacks who must be designed for this protection so that
// they are the source of truth and so that the compiler catches type issues
// when compiling (versus when the callback is attached).
type (
	BlockingPrologCallbackType = func(**RequestContext) (BlockingEpilogCallbackType, error)
	BlockingEpilogCallbackType = func(*error)

	NonBlockingPrologCallbackType = func(**RequestContext) (NonBlockingEpilogCallbackType, error)
	NonBlockingEpilogCallbackType = func()

	WAFPrologCallbackType = BlockingPrologCallbackType
	WAFEpilogCallbackType = BlockingEpilogCallbackType

	BodyWAFPrologCallbackType = WAFPrologCallbackType
	BodyWAFEpilogCallbackType = WAFEpilogCallbackType

	IdentifyUserPrologCallbackType = func(**RequestContext, *map[string]string) (BlockingEpilogCallbackType, error)

	ResponseMonitoringPrologCallbackType = func(**RequestContext, *types.ResponseFace) (NonBlockingEpilogCallbackType, error)
)

// Static assert that RequestContext implements the SDK Event Recorder Getter
// interface.
var _ protectioncontext.EventRecorderGetter = (*RequestContext)(nil)

func (c *RequestContext) EventRecorder() protectioncontext.EventRecorder { return c }

type requestReader struct {
	types.RequestReader

	// clientIP is the actual IP address of the client performing the request.
	clientIP net.IP

	// requestParams is the set of HTTP request parameters taken from the HTTP
	// request. The map key is the source (eg. json, query, multipart-form, etc.)
	// so that we can report it and make it clearer to understand where the value
	// comes from.
	requestParams types.RequestParamMap

	// bodyReadBuffer is the buffers body reads
	bodyReadBuffer bytes.Buffer
}

func (r *requestReader) ClientIP() net.IP { return r.clientIP }

func (r *requestReader) Params() types.RequestParamMap {
	params := r.RequestReader.Params()
	if len(params) == 0 {
		return r.requestParams
	} else if len(r.requestParams) == 0 {
		return params
	}

	res := make(types.RequestParamMap, len(params)+len(r.requestParams))
	for n, v := range params {
		res[n] = v
	}
	return res
}

func (c *RequestContext) AddAttackEvent(attack *event.AttackEvent) {
	c.events.AddAttackEvent(attack)
}

func (c *RequestContext) TrackEvent(event string) protectioncontext.CustomEvent {
	return c.events.AddCustomEvent(event)
}

func (c *RequestContext) TrackUserSignup(id map[string]string) {
	c.events.AddUserSignup(id, c.RequestReader.ClientIP())
}

func (c *RequestContext) TrackUserAuth(id map[string]string, success bool) {
	c.events.AddUserAuth(id, c.RequestReader.ClientIP(), success)
}

func (c *RequestContext) IdentifyUser(id map[string]string) error {
	c.events.Identify(id)
	return c.userSecurityResponse(id)
}

// Static assert that the SDK interface is implemented.
var _ protectioncontext.EventRecorder = &RequestContext{}

func FromContext(ctx context.Context) *RequestContext {
	c, _ := protectioncontext.FromContext(ctx).(*RequestContext)
	return c
}

func FromGLS() *RequestContext {
	ctx := sqgls.Get()
	if ctx == nil {
		return nil
	}
	return ctx.(*RequestContext)
}

func NewRequestContext(ctx context.Context, agent protectioncontext.AgentFace, w types.ResponseWriter, r types.RequestReader) (*RequestContext, context.Context, context.CancelFunc) {
	if agent.IsPathAllowed(r.URL().Path) {
		return nil, nil, nil
	}

	clientIP := r.ClientIP()
	if clientIP == nil {
		cfg := agent.Config()
		clientIP = ClientIP(r.RemoteAddr(), r.Headers(), cfg.PrioritizedIPHeader(), cfg.PrioritizedIPHeaderFormat())
	}

	if agent.IsIPAllowed(clientIP) {
		return nil, nil, nil
	}

	reqCtx, cancelHandlerContextFunc := context.WithCancel(ctx)

	rr := &requestReader{
		clientIP:      clientIP,
		RequestReader: r,
		requestParams: make(types.RequestParamMap),
	}

	protCtx := &RequestContext{
		RequestContext:           protectioncontext.NewRequestContext(agent),
		ResponseWriter:           w,
		RequestReader:            rr,
		cancelHandlerContextFunc: cancelHandlerContextFunc,
		// Keep a reference to the request param map to be able to add more params
		// to it.
		requestReader: rr,
	}
	return protCtx, reqCtx, cancelHandlerContextFunc
}

// When a non-nil error is returned, the request handler shouldn't be called
// and the request should be stopped immediately by closing the RequestContext
// and returning.
func (c *RequestContext) Before() (err error) {
	// Set the current goroutine local storage to this request storage to be able
	// to retrieve it from lower-level functions.
	sqgls.Set(c)

	c.addSecurityHeaders()

	if err := c.isIPBlocked(); err != nil {
		return err
	}
	if err := c.ipSecurityResponse(); err != nil {
		return err
	}
	if err := c.waf(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func (c *RequestContext) isIPBlocked() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) waf() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) bodyWAF() error { /* dynamically instrumented */ return nil }

//go:noinline
func (c *RequestContext) addSecurityHeaders() { /* dynamically instrumented */ }

//go:noinline
func (c *RequestContext) ipSecurityResponse() error { /* dynamically instrumented */ return nil }

type canceledHandlerContextError struct{}

func (canceledHandlerContextError) Error() string { return "canceled handler context" }

//go:noinline
func (c *RequestContext) After() (err error) {
	if c.isContextHandlerCanceled() {
		// The context was canceled by an in-handler protection, return an error
		// in order to fully abort the framework.
		return canceledHandlerContextError{}
	}

	return nil
}

func (c *RequestContext) userSecurityResponse(userID map[string]string) error { return nil }

// CancelHandlerContext cancels the request handler context in order to stop its
// execution and abort every ongoing operation and goroutine it may be doing.
// Since the handler should return at some point, the After() protection method
// will take care of applying the blocking response.
// This method can be called by multiple goroutines simultaneously.
func (c *RequestContext) CancelHandlerContext() {
	c.contextHandlerCanceledLock.Lock()
	defer c.contextHandlerCanceledLock.Unlock()
	c.contextHandlerCanceled = true
}

func (c *RequestContext) isContextHandlerCanceled() bool {
	c.contextHandlerCanceledLock.RLock()
	defer c.contextHandlerCanceledLock.RUnlock()
	return c.contextHandlerCanceled

}

type closedRequestContext struct {
	response types.ResponseFace
	request  types.RequestReader
	events   event.Recorded
}

var _ types.ClosedRequestContextFace = (*closedRequestContext)(nil)

func (c *closedRequestContext) Events() event.Recorded {
	return c.events
}

func (c *closedRequestContext) Request() types.RequestReader {
	return c.request
}

func (c *closedRequestContext) Response() types.ResponseFace {
	return c.response
}

func (c *RequestContext) Close(response types.ResponseFace) error {
	// Make sure to clear the goroutine local storage to avoid keeping it if some
	// memory pools are used under the hood.
	// TODO: enforce this by design of the gls instrumentation
	defer sqgls.Set(nil)

	// Copy everything we need here as it is not safe to keep then after the
	// request is done because of memory pools reusing them.
	c.monitorObservedResponse(response)
	return c.RequestContext.Close(&closedRequestContext{
		response: response,
		request:  copyRequest(c.RequestReader),
		events:   c.events.CloseRecord(),
	})
}

func copyRequest(reader types.RequestReader) types.RequestReader {
	return &handledRequest{
		headers:    reader.Headers(),
		method:     reader.Method(),
		url:        reader.URL(),
		requestURI: reader.RequestURI(),
		host:       reader.Host(),
		remoteAddr: reader.RemoteAddr(),
		isTLS:      reader.IsTLS(),
		userAgent:  reader.UserAgent(),
		referer:    reader.Referer(),
		form:       reader.Form(),
		postForm:   reader.PostForm(),
		clientIP:   reader.ClientIP(),
		params:     reader.Params(),
		body:       reader.Body(),
	}
}

type handledRequest struct {
	headers    http.Header
	method     string
	url        *url.URL
	requestURI string
	host       string
	remoteAddr string
	isTLS      bool
	userAgent  string
	referer    string
	form       url.Values
	postForm   url.Values
	clientIP   net.IP
	params     types.RequestParamMap
	body       []byte
}

func (h *handledRequest) Headers() http.Header          { return h.headers }
func (h *handledRequest) Method() string                { return h.method }
func (h *handledRequest) URL() *url.URL                 { return h.url }
func (h *handledRequest) RequestURI() string            { return h.requestURI }
func (h *handledRequest) Host() string                  { return h.host }
func (h *handledRequest) RemoteAddr() string            { return h.remoteAddr }
func (h *handledRequest) IsTLS() bool                   { return h.isTLS }
func (h *handledRequest) UserAgent() string             { return h.userAgent }
func (h *handledRequest) Referer() string               { return h.referer }
func (h *handledRequest) Form() url.Values              { return h.form }
func (h *handledRequest) PostForm() url.Values          { return h.postForm }
func (h *handledRequest) ClientIP() net.IP              { return h.clientIP }
func (h *handledRequest) Params() types.RequestParamMap { return h.params }
func (h *handledRequest) Body() []byte                  { return h.body }
func (h *handledRequest) Header(header string) (value *string) {
	headers := h.headers
	if headers == nil {
		return nil
	}
	v := headers[textproto.CanonicalMIMEHeaderKey(header)]
	if len(v) == 0 {
		return nil
	}
	return &v[0]
}

// Write the default blocking response. This method only write the response, it
// doesn't block nor cancel the handler context. Users of this method must
// handle their
//go:noinline
func (c *RequestContext) WriteDefaultBlockingResponse() { /* dynamically instrumented */ }

//go:noinline
func (c *RequestContext) monitorObservedResponse(response types.ResponseFace) { /* dynamically instrumented */ }

// WrapRequest is a helper method to prepare an http.Request with its
// new context, the protection context, and a body buffer.
func (c *RequestContext) WrapRequest(ctx context.Context, r *http.Request) *http.Request {
	r = r.WithContext(context.WithValue(ctx, protectioncontext.ContextKey, c))
	if r.Body != nil {
		r.Body = c.wrapBody(r.Body)
	}
	return r
}

func (c *RequestContext) wrapBody(body io.ReadCloser) io.ReadCloser {
	return rawBodyWAF{
		ReadCloser: body,
		c:          c,
	}
}

// AddRequestParam adds a new request parameter to the set. Request parameters
// are taken from the HTTP request and parsed into a Go value. It can be the
// result of a JSON parsing, query-string parsing, etc. The source allows to
// specify where it was taken from.
func (c *RequestContext) AddRequestParam(name string, param interface{}) {
	params := c.requestReader.requestParams[name]
	c.requestReader.requestParams[name] = append(params, param)
}

type rawBodyWAF struct {
	io.ReadCloser
	c *RequestContext
}

// Read buffers what has been read and ultimately calls the WAF on EOF.
func (t rawBodyWAF) Read(p []byte) (n int, err error) {
	n, err = t.ReadCloser.Read(p)
	if n > 0 {
		t.c.requestReader.bodyReadBuffer.Write(p[:n])
	}
	fmt.Println(err)
	if err == io.EOF {
		if wafErr := t.c.bodyWAF(); wafErr != nil {
			err = wafErr
		}
	}
	return
}

//go:noinline
func (c *RequestContext) onEOF() error { return nil /* dynamically instrumented */ }

func ClientIP(remoteAddr string, headers http.Header, prioritizedIPHeader string, prioritizedIPHeaderFormat string) net.IP {
	var privateIP net.IP
	check := func(value string) net.IP {
		for _, ip := range strings.Split(value, ",") {
			ipStr := strings.Trim(ip, " ")
			ipStr, _ = splitHostPort(ipStr)
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil
			}

			if isGlobal(ip) {
				return ip
			}

			if privateIP == nil && !ip.IsLoopback() && isPrivate(ip) {
				privateIP = ip
			}
		}
		return nil
	}

	if prioritizedIPHeader != "" {
		if value := headers.Get(prioritizedIPHeader); value != "" {
			if prioritizedIPHeaderFormat != "" {
				parsed, err := parseClientIPHeaderHeaderValue(prioritizedIPHeaderFormat, value)
				if err == nil {
					// Parsing ok, keep its returned value.
					value = parsed
				} else {
					// An error occurred while parsing the header value, so ignore it.
					value = ""
				}
			}

			if value != "" {
				if ip := check(value); ip != nil {
					return ip
				}
			}
		}
	}

	for _, key := range config.IPRelatedHTTPHeaders {
		value := headers.Get(key)
		if ip := check(value); ip != nil {
			return ip
		}
	}

	remoteIPStr, _ := splitHostPort(remoteAddr)
	if remoteIPStr == "" {
		if privateIP != nil {
			return privateIP
		}
		return nil
	}

	if remoteIP := net.ParseIP(remoteIPStr); remoteIP != nil && (privateIP == nil || isGlobal(remoteIP)) {
		return remoteIP
	}
	return privateIP
}

func isGlobal(ip net.IP) bool {
	if ipv4 := ip.To4(); ipv4 != nil && config.IPv4PublicNetwork.Contains(ipv4) {
		return false
	}
	return !isPrivate(ip)
}

func isPrivate(ip net.IP) bool {
	var privateNetworks []*net.IPNet
	// We cannot rely on `len(IP)` to know what type of IP address this is.
	// `net.ParseIP()` or `net.IPv4()` can return internal 16-byte representations
	// of an IP address even if it is an IPv4. So the trick is to use `IP.To4()`
	// which returns nil if the address in not an IPv4 address.
	if ipv4 := ip.To4(); ipv4 != nil {
		privateNetworks = config.IPv4PrivateNetworks
	} else {
		privateNetworks = config.IPv6PrivateNetworks
	}

	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// SplitHostPort splits a network address of the form `host:port` or
// `[host]:port` into `host` and `port`.
func splitHostPort(addr string) (host string, port string) {
	i := strings.LastIndex(addr, "]:")
	if i != -1 {
		// ipv6
		return strings.Trim(addr[:i+1], "[]"), addr[i+2:]
	}

	i = strings.LastIndex(addr, ":")
	if i == -1 {
		// not an address with a port number
		return addr, ""
	}
	return addr[:i], addr[i+1:]
}

func parseClientIPHeaderHeaderValue(format, value string) (string, error) {
	// Hard-coded HA Proxy format for now: `%ci:%cp...` so we expect the value to
	// start with the client IP in hexadecimal format (eg. 7F000001) separated by
	// the client port number with a semicolon `:`.
	sep := strings.IndexRune(value, ':')
	if sep == -1 {
		return "", errors.Errorf("unexpected IP address value `%s`", value)
	}

	clientIPHexStr := value[:sep]
	// Optimize for the best case: there will be an IP address, so allocate size
	// for at least an IPv4 address.
	clientIPBuf := make([]byte, 0, net.IPv4len)
	_, err := fmt.Sscanf(clientIPHexStr, "%x", &clientIPBuf)
	if err != nil {
		return "", errors.Wrap(err, "could not parse the IP address value")
	}

	switch len(clientIPBuf) {
	case net.IPv4len, net.IPv6len:
		return net.IP(clientIPBuf).String(), nil
	default:
		return "", errors.Errorf("unexpected IP address value `%s`", clientIPBuf)
	}
}

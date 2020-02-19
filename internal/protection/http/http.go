// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/event"
	protection_context "github.com/sqreen/go-agent/internal/protection/context"
	"github.com/sqreen/go-agent/internal/protection/http/types"
)

type RequestContext struct {
	*protection_context.RequestContext
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
}

// Static assert that RequestContext implements the SDK Event Recorder Getter
// interface.
var _ protection_context.EventRecorderGetter = (*RequestContext)(nil)

func (c *RequestContext) EventRecorder() protection_context.EventRecorder {
	return c
}

type requestReader struct {
	types.RequestReader
	clientIP net.IP
}

func (r requestReader) ClientIP() net.IP { return r.clientIP }

func (c *RequestContext) AddAttackEvent(attack *event.AttackEvent) {
	c.events.AddAttackEvent(attack)
}

func (c *RequestContext) TrackEvent(event string) protection_context.CustomEvent {
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
var _ protection_context.EventRecorder = &RequestContext{}

func FromContext(ctx context.Context) *RequestContext {
	c, _ := protection_context.FromContext(ctx).(*RequestContext)
	return c
}

func NewRequestContext(agent protection_context.AgentFace, w types.ResponseWriter, r types.RequestReader, cancelHandlerContextFunc context.CancelFunc) *RequestContext {
	if agent == nil {
		return nil
	}
	if r.ClientIP() == nil {
		cfg := agent.Config()
		r = requestReader{
			clientIP:      ClientIP(r.RemoteAddr(), r.Headers(), cfg.PrioritizedIPHeader(), cfg.PrioritizedIPHeaderFormat()),
			RequestReader: r,
		}
	}
	if agent.IsIPWhitelisted(r.ClientIP()) {
		return nil
	}
	ctx := &RequestContext{
		RequestContext:           protection_context.NewRequestContext(agent),
		ResponseWriter:           w,
		RequestReader:            r,
		cancelHandlerContextFunc: cancelHandlerContextFunc,
	}
	return ctx
}

// When a non-nil error is returned, the request handler shouldn't be called
// and the request should be stopped immediately by closing the RequestContext
// and returning.
func (c *RequestContext) Before() (err error) {
	c.addSecurityHeaders()
	if err := c.ipSecurityResponse(); err != nil {
		return err
	}
	if err := c.waf(); err != nil {
		return err
	}
	return nil
}

//go:noinline
func (c *RequestContext) waf() error { /* dynamically instrumented */ return nil }

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
	// Do not assume values are safe to keep after the request is done, as they
	// might be put back in a shared pool.
	c.monitorObservedResponse(response)
	return c.RequestContext.Close(&closedRequestContext{
		response: response,
		request:  copyRequest(c.RequestReader),
		events:   c.events.CloseRecord(),
	})
}

func copyRequest(reader types.RequestReader) types.RequestReader {
	return &handledRequest{
		headers:         reader.Headers(),
		method:          reader.Method(),
		url:             reader.URL(),
		requestURI:      reader.RequestURI(),
		host:            reader.Host(),
		remoteAddr:      reader.RemoteAddr(),
		isTLS:           reader.IsTLS(),
		userAgent:       reader.UserAgent(),
		referer:         reader.Referer(),
		form:            reader.Form(),
		postForm:        reader.PostForm(),
		clientIP:        reader.ClientIP(),
		frameworkParams: reader.FrameworkParams(),
	}
}

type handledRequest struct {
	headers         http.Header
	method          string
	url             *url.URL
	requestURI      string
	host            string
	remoteAddr      string
	isTLS           bool
	userAgent       string
	referer         string
	form            url.Values
	postForm        url.Values
	clientIP        net.IP
	frameworkParams url.Values
}

func (h *handledRequest) Headers() http.Header        { return h.headers }
func (h *handledRequest) Method() string              { return h.method }
func (h *handledRequest) URL() *url.URL               { return h.url }
func (h *handledRequest) RequestURI() string          { return h.requestURI }
func (h *handledRequest) Host() string                { return h.host }
func (h *handledRequest) RemoteAddr() string          { return h.remoteAddr }
func (h *handledRequest) IsTLS() bool                 { return h.isTLS }
func (h *handledRequest) UserAgent() string           { return h.userAgent }
func (h *handledRequest) Referer() string             { return h.referer }
func (h *handledRequest) Form() url.Values            { return h.form }
func (h *handledRequest) PostForm() url.Values        { return h.postForm }
func (h *handledRequest) ClientIP() net.IP            { return h.clientIP }
func (h *handledRequest) FrameworkParams() url.Values { return h.frameworkParams }
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

func ClientIP(remoteAddr string, headers http.Header, prioritizedIPHeader string, prioritizedIPHeaderFormat string) net.IP {
	var privateIP net.IP
	check := func(value string) net.IP {
		for _, ip := range strings.Split(value, ",") {
			ipStr := strings.Trim(ip, " ")
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
	remoteIPStr, _ := splitHostPort(remoteAddr) // FIXME: replace by net.SplitHostPort?
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

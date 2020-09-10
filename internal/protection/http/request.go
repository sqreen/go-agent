// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package http

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/protection/http/types"
)

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

func (r *requestReader) Body() []byte { return r.bodyReadBuffer.Bytes() }

func (r *requestReader) ClientIP() net.IP { return r.clientIP }

func (r *requestReader) Params() types.RequestParamMap {
	params := r.RequestReader.Params()
	if len(params) == 0 {
		return r.requestParams
	}

	if len(r.requestParams) == 0 {
		return params
	}

	res := make(types.RequestParamMap, len(params)+len(r.requestParams))
	for n, v := range params {
		res[n] = v
	}
	for n, v := range r.requestParams {
		res[n] = v
	}
	return res
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

	if err == io.EOF {
		if wafErr := t.c.bodyWAF(); wafErr != nil {
			// Return 0 and the sqreen error so that the caller doesn't take anything
			// into account.
			n = 0
			err = wafErr
		}
	}
	return
}

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
	queryForm  url.Values
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
func (h *handledRequest) QueryForm() url.Values         { return h.queryForm }
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
		queryForm:  reader.QueryForm(),
		postForm:   reader.PostForm(),
		clientIP:   reader.ClientIP(),
		params:     reader.Params(),
		body:       reader.Body(),
	}
}

type closedRequestContext struct {
	response types.ResponseFace
	request  types.RequestReader
	events   event.Recorded
}

var _ types.ClosedRequestContextFace = (*closedRequestContext)(nil)

func (c *closedRequestContext) Events() event.Recorded       { return c.events }
func (c *closedRequestContext) Request() types.RequestReader { return c.request }
func (c *closedRequestContext) Response() types.ResponseFace { return c.response }

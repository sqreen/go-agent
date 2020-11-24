// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package grpc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"time"

	"github.com/sqreen/go-agent/internal/event"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqgls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type UnaryRPCProtectionContext struct {
	types.RootProtectionContext

	RequestReader  types.RequestReader
	ResponseWriter types.ResponseWriter

	events event.Record

	start    time.Time
	clientIP net.IP
}

func (p *UnaryRPCProtectionContext) AddRequestParam(name string, v interface{}) {
	panic("implement me")
}

func (p *UnaryRPCProtectionContext) ClientIP() net.IP {
	return p.clientIP
}

type requestReader struct {
	clientIP   net.IP
	req        interface{}
	md         metadata.MD
	method     string
	url        *url.URL
	host       string
	remoteAddr string
	userAgent  string
}

func (r *requestReader) Header(header string) (value *string) {
	values := r.md.Get(header)
	if len(values) == 0 {
		return nil
	}
	return &values[0]
}

func (r *requestReader) Headers() http.Header {
	return http.Header(r.md)
}

func (r *requestReader) Method() string {
	return r.method
}

func (r *requestReader) URL() *url.URL {
	return r.url
}

func (r *requestReader) RequestURI() string {
	return r.url.RequestURI()
}

func (r *requestReader) Host() string {
	return r.host
}

func (r *requestReader) RemoteAddr() string {
	return r.remoteAddr
}

func (r *requestReader) IsTLS() bool {
	return true
}

func (r *requestReader) UserAgent() string {
	return r.userAgent
}

func (r *requestReader) Referer() string {
	return ""
}

func (r *requestReader) QueryForm() url.Values {
	return nil
}

func (r *requestReader) PostForm() url.Values {
	return nil
}

func (r *requestReader) ClientIP() net.IP {
	return r.clientIP
}

func (r *requestReader) Params() types.RequestParamMap {
	return types.RequestParamMap{"request": types.RequestParamValueSlice{r.req}}
}

func (r *requestReader) Body() []byte {
	return nil
}

func NewUnaryRPCProtectionContext(ctx types.RootProtectionContext, req interface{}, info *grpc.UnaryServerInfo) *UnaryRPCProtectionContext {
	if ctx == nil {
		return nil
	}

	goCtx := ctx.Context()
	md, ok := metadata.FromIncomingContext(goCtx)
	if !ok {
		return nil
	}

	peer, ok := peer.FromContext(goCtx)
	if !ok {
		return nil
	}

	cfg := ctx.Config()
	clientIP := http_protection.ClientIP(peer.Addr.String(), http.Header(md), cfg.HTTPClientIPHeader(), cfg.HTTPClientIPHeaderFormat())

	host := md[":authority"][0]

	url, err := url.Parse(host + info.FullMethod)
	if err != nil {
		return nil
	}

	rr := &requestReader{
		clientIP:   clientIP,
		req:        req,
		md:         md,
		method:     info.FullMethod,
		url:        url,
		host:       url.Host,
		remoteAddr: peer.Addr.String(),
		userAgent:  md["user-agent"][0],
	}

	if ctx.IsIPAllowed(clientIP) {
		return nil
	}

	p := &UnaryRPCProtectionContext{
		RootProtectionContext: ctx,
		RequestReader:         rr,
		clientIP:              clientIP,
	}
	return p
}

func (p *UnaryRPCProtectionContext) Before() error {
	p.start = time.Now()

	// Set the current goroutine local storage to this request storage to be able
	// to retrieve it from lower-level functions.
	sqgls.Set(p)
	// TODO: IP denylsit
	// TODO: IP Sec Response
	// TODO: WAF
	return nil
}

func (p *UnaryRPCProtectionContext) After() error {
	if p.Context().Err() == context.Canceled {
		// The context was canceled by an in-handler protection, return an error
		// in order to fully abort the framework.
		return errors.New("canceled request context")
	}

	return nil
}

func (p *UnaryRPCProtectionContext) HandleAttack(block bool, attack *event.AttackEvent) (blocked bool) {
	if block {
		defer p.CancelContext()
		blocked = true
	}

	if attack != nil {
		p.events.AddAttackEvent(attack)
	}

	return blocked
}

func (p *UnaryRPCProtectionContext) Close(resp types.ResponseFace) {
	// Compute the request duration
	duration := time.Since(p.start)

	// Make sure to clear the goroutine local storage to avoid keeping it if some
	// memory pools are used under the hood.
	// TODO: enforce this by design of the gls instrumentation
	defer sqgls.Set(nil)

	// Copy everything we need here as it is not safe to keep then after the
	// request is done because of memory pools reusing them.
	// TODO: monitor response
	p.RootProtectionContext.Close(&closedProtectionContext{
		request:    copyRequest(p.RequestReader),
		response:   resp,
		events:     p.events.CloseRecord(),
		start:      p.start,
		duration:   duration,
		sqreenTime: p.SqreenTime().Duration(),
	})
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

type closedProtectionContext struct {
	response   types.ResponseFace
	request    types.RequestReader
	events     event.Recorded
	start      time.Time
	duration   time.Duration
	sqreenTime time.Duration
}

var _ types.ClosedProtectionContextFace = (*closedProtectionContext)(nil)

func (c *closedProtectionContext) Events() event.Recorded       { return c.events }
func (c *closedProtectionContext) Request() types.RequestReader { return c.request }
func (c *closedProtectionContext) Response() types.ResponseFace { return c.response }
func (c *closedProtectionContext) Start() time.Time             { return c.start }
func (c *closedProtectionContext) Duration() time.Duration      { return c.duration }
func (c *closedProtectionContext) SqreenTime() time.Duration    { return c.sqreenTime }

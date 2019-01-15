package agent

import (
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/agent/config"
)

type HTTPRequest interface {
	ClientIP() string
	StdRequest() *http.Request
}

type HTTPRequestContext struct {
	// Copy of the request, safe to be asynchronously read, even after the request
	// was terminated.
	request    HTTPRequest
	eventsLock sync.Mutex
	events     []*HTTPRequestEvent
}

func NewHTTPRequestContext(req HTTPRequest) *HTTPRequestContext {
	return &HTTPRequestContext{
		request: req,
	}
}

type HTTPRequestEvent struct {
	method     string
	event      string
	properties EventPropertyMap
	timestamp  time.Time
}

type EventPropertyMap map[string]interface{}

func (ctx *HTTPRequestContext) Track(event string) *HTTPRequestEvent {
	evt := &HTTPRequestEvent{
		method:     "track",
		event:      event,
		properties: nil,
		timestamp:  time.Now(),
	}
	ctx.addEvent(evt)
	return evt
}

func (ctx *HTTPRequestContext) Close() {
	addEvent(newHTTPRequestRecord(ctx))
}

func (ctx *HTTPRequestContext) addEvent(event *HTTPRequestEvent) {
	ctx.eventsLock.Lock()
	defer ctx.eventsLock.Unlock()
	ctx.events = append(ctx.events, event)
}

func (e *HTTPRequestEvent) WithTimestamp(t time.Time) *HTTPRequestEvent {
	e.timestamp = t
	return e
}

func (e *HTTPRequestEvent) WithProperties(p EventPropertyMap) *HTTPRequestEvent {
	e.properties = p
	return e
}

func (e *HTTPRequestEvent) GetTime() time.Time {
	return e.timestamp
}

func (e *HTTPRequestEvent) GetName() string {
	return e.method
}

func (e *HTTPRequestEvent) GetArgs() api.ListValue {
	opts := &api.RequestRecord_Observed_SDKEvent_Options{
		Properties: &api.Struct{e.properties},
	}
	return api.ListValue([]interface{}{e.event, opts})
}

func (e *HTTPRequestEvent) Proto() proto.Message {
	return api.NewRequestRecord_Observed_SDKEventFromFace(e)
}

type httpRequestRecord struct {
	ctx         *HTTPRequestContext
	rulespackID string
}

func newHTTPRequestRecord(event *HTTPRequestContext) *httpRequestRecord {
	return &httpRequestRecord{
		ctx: event,
	}
}

func (r *httpRequestRecord) GetVersion() string {
	return api.RequestRecordVersion
}

func (r *httpRequestRecord) GetRulespackId() string {
	return r.rulespackID
}

func (r *httpRequestRecord) SetRulespackId(rulespackId string) {
	r.rulespackID = rulespackId
}

func (r *httpRequestRecord) GetClientIp() string {
	req := r.ctx.request.StdRequest()

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

	if prioritizedHeader := config.HTTPClientIPHeader(); prioritizedHeader != "" {
		if value := req.Header.Get(prioritizedHeader); value != "" {
			if ip := check(value); ip != nil {
				return ip.String()
			}
		}
	}

	for _, key := range config.IPRelatedHTTPHeaders {
		value := req.Header.Get(key)
		if ip := check(value); ip != nil {
			return ip.String()
		}
	}

	remoteIP, _ := parseAddr(req.RemoteAddr)
	if remoteIP == "" {
		if privateIP != nil {
			return privateIP.String()
		}
		return ""
	}

	if privateIP == nil || isGlobal(net.ParseIP(remoteIP)) {
		return remoteIP
	}
	return privateIP.String()
}

func (r *httpRequestRecord) GetRequest() api.RequestRecord_Request {
	req := r.ctx.request.StdRequest()

	headers := make([]api.RequestRecord_Request_Header, 0, len(req.Header))
	for _, header := range config.TrackedHTTPHeaders {
		if value := req.Header.Get(header); value != "" {
			headers = append(headers, api.RequestRecord_Request_Header{
				Key:   header,
				Value: value,
			})
		}
	}

	remoteIP, remotePort := parseAddr(req.RemoteAddr)
	_, hostPort := parseAddr(req.Host)

	var scheme string
	if req.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}

	requestId := req.Header.Get("X-Request-Id")
	if requestId == "" {
		uuid, err := uuid.NewRandom()
		if err != nil {
			logger.Error("could not generate a request id ", err)
			requestId = ""
		}
		requestId = hex.EncodeToString(uuid[:])
	}

	// FIXME: create it from an interface for compile-time errors.
	return api.RequestRecord_Request{
		Rid:        requestId,
		Headers:    headers,
		Verb:       req.Method,
		RawPath:    req.RequestURI,
		Path:       req.URL.Path,
		Host:       req.Host,
		Port:       hostPort,
		RemoteIp:   remoteIP,
		RemotePort: remotePort,
		Scheme:     scheme,
		UserAgent:  req.UserAgent(),
		Referer:    req.Referer(),
	}
}

func (r *httpRequestRecord) GetResponse() api.RequestRecord_Response {
	return api.RequestRecord_Response{}
}

func (r *httpRequestRecord) GetObserved() api.RequestRecord_Observed {
	events := make([]*api.RequestRecord_Observed_SDKEvent, 0, len(r.ctx.events))
	for _, event := range r.ctx.events {
		events = append(events, api.NewRequestRecord_Observed_SDKEventFromFace(event))
	}

	return api.RequestRecord_Observed{
		Sdk: events,
	}
}

func (r *httpRequestRecord) Proto() proto.Message {
	return api.NewRequestRecordFromFace(r)
}

func isGlobal(ip net.IP) bool {
	if len(ip) == 4 && config.IPv4PublicNetwork.Contains(ip) {
		return false
	}
	return !isPrivate(ip)
}

func isPrivate(ip net.IP) bool {
	for _, network := range config.IPPrivateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseAddr(addr string) (host string, port string) {
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

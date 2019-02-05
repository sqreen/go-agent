package agent

import (
	"encoding/hex"
	"encoding/json"
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

type HTTPRequestRecord struct {
	// Copy of the request, safe to be asynchronously read, even after the request
	// was terminated.
	request    *http.Request
	eventsLock sync.Mutex
	events     []*HTTPRequestEvent
}

func NewHTTPRequestRecord(req *http.Request) *HTTPRequestRecord {
	return &HTTPRequestRecord{
		request: req,
	}
}

type HTTPRequestEvent struct {
	method          string
	event           string
	properties      EventPropertyMap
	userIdentifiers EventUserIdentifiersMap
	timestamp       time.Time
}

type userEventFace interface {
	isUserEvent()
	metricEntry
}

type userEvent struct {
	userIdentifiers EventUserIdentifiersMap
	timestamp       time.Time
	ip              string
}

type authUserEvent struct {
	*userEvent
	loginSuccess bool
}

func (_ *authUserEvent) isUserEvent() {}

func (e *authUserEvent) bucketID() (string, error) {
	k := &userMetricKey{
		id: e.userEvent.userIdentifiers,
		ip: e.userEvent.ip,
	}
	return k.bucketID()
}

type userMetricKey struct {
	id EventUserIdentifiersMap
	ip string
}

func (k *userMetricKey) bucketID() (string, error) {
	var keys [][]interface{}
	for prop, val := range k.id {
		keys = append(keys, []interface{}{prop, val})
	}
	v := struct {
		Keys [][]interface{} `json:"keys"`
		IP   string          `json:"ip"`
	}{
		Keys: keys,
		IP:   k.ip,
	}
	buf, err := json.Marshal(&v)
	return string(buf), err
}

type signupUserEvent struct {
	*userEvent
}

func (e *signupUserEvent) bucketID() (string, error) {
	k := &userMetricKey{
		id: e.userEvent.userIdentifiers,
		ip: e.userEvent.ip,
	}
	return k.bucketID()
}

func (_ *signupUserEvent) isUserEvent() {}

type EventPropertyMap map[string]string

type EventUserIdentifiersMap map[string]string

func (ctx *HTTPRequestRecord) TrackEvent(event string) *HTTPRequestEvent {
	evt := &HTTPRequestEvent{
		method:    "track",
		event:     event,
		timestamp: time.Now(),
	}
	ctx.addEvent(evt)
	return evt
}

func (ctx *HTTPRequestRecord) TrackIdentify(id EventUserIdentifiersMap) *HTTPRequestEvent {
	evt := &HTTPRequestEvent{
		method:          "identify",
		timestamp:       time.Now(),
		userIdentifiers: id,
	}
	ctx.addEvent(evt)
	return evt
}

func (ctx *HTTPRequestRecord) TrackAuth(loginSuccess bool, id EventUserIdentifiersMap) {
	if len(id) == 0 {
		logger.Warn("TrackAuth(): user id is nil or empty")
		return
	}

	event := &authUserEvent{
		loginSuccess: loginSuccess,
		userEvent: &userEvent{
			ip:              getClientIP(ctx.request),
			userIdentifiers: id,
			timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *HTTPRequestRecord) TrackSignup(id EventUserIdentifiersMap) {
	if len(id) == 0 {
		logger.Warn("TrackSignup(): user id is nil or empty")
		return
	}

	event := &signupUserEvent{
		userEvent: &userEvent{
			ip:              getClientIP(ctx.request),
			userIdentifiers: id,
			timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *HTTPRequestRecord) Close() {
	addTrackEvent(newHTTPRequestRecord(ctx))
}

func (ctx *HTTPRequestRecord) addEvent(event *HTTPRequestEvent) {
	ctx.eventsLock.Lock()
	defer ctx.eventsLock.Unlock()
	ctx.events = append(ctx.events, event)
}

func (ctx *HTTPRequestRecord) addUserEvent(event userEventFace) {
	addUserEvent(event)
}

func (e *HTTPRequestEvent) WithTimestamp(t time.Time) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.timestamp = t
	return e
}

func (e *HTTPRequestEvent) WithProperties(p EventPropertyMap) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.properties = p
	return e
}

func (e *HTTPRequestEvent) WithUserIdentifier(id EventUserIdentifiersMap) *HTTPRequestEvent {
	if e == nil {
		return nil
	}
	e.userIdentifiers = id
	return e
}

func (e *HTTPRequestEvent) GetTime() time.Time {
	return e.timestamp
}

func (e *HTTPRequestEvent) GetName() string {
	return e.method
}

func (e *HTTPRequestEvent) GetArgs() api.ListValue {
	opts := api.NewRequestRecord_Observed_SDKEvent_OptionsFromFace(e)
	return api.ListValue([]interface{}{e.event, opts})
}

func (e *HTTPRequestEvent) GetProperties() *api.Struct {
	return &api.Struct{e.properties}
}

func (e *HTTPRequestEvent) GetUserIdentifiers() *api.Struct {
	return &api.Struct{e.userIdentifiers}
}

func (e *HTTPRequestEvent) Proto() proto.Message {
	return api.NewRequestRecord_Observed_SDKEventFromFace(e)
}

type httpRequestRecord struct {
	ctx         *HTTPRequestRecord
	rulespackID string
}

func newHTTPRequestRecord(event *HTTPRequestRecord) *httpRequestRecord {
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
	return getClientIP(r.ctx.request)
}

func getClientIP(req *http.Request) string {
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
	req := r.ctx.request

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

package internal

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/types"
)

type HTTPRequestRecord struct {
	// Copy of the request, safe to be asynchronously read, even after the request
	// was terminated.
	request      *http.Request
	eventsLock   sync.Mutex
	events       []*HTTPRequestEvent
	identifyOnce sync.Once
	agent        *Agent
	shouldSend   bool
}

func (a *Agent) newHTTPRequestRecord(req *http.Request) *HTTPRequestRecord {
	return &HTTPRequestRecord{
		request: req,
		agent:   a,
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

const (
	sdkMethodIdentify = "identify"
	sdkMethodTrack    = "track"
)

func (ctx *HTTPRequestRecord) NewCustomEvent(event string) types.CustomEvent {
	evt := &HTTPRequestEvent{
		method:    sdkMethodTrack,
		event:     event,
		timestamp: time.Now(),
	}
	ctx.addEvent(evt)
	return evt
}

func (ctx *HTTPRequestRecord) Identify(id map[string]string) {
	ctx.identifyOnce.Do(func() {
		evt := &HTTPRequestEvent{
			method:          sdkMethodIdentify,
			timestamp:       time.Now(),
			userIdentifiers: id,
		}
		ctx.addSilentEvent(evt)
	})
}

func (ctx *HTTPRequestRecord) NewUserAuth(id map[string]string, loginSuccess bool) {
	if len(id) == 0 {
		ctx.agent.logger.Warn("TrackAuth(): user id is nil or empty")
		return
	}

	event := &authUserEvent{
		loginSuccess: loginSuccess,
		userEvent: &userEvent{
			ip:              getClientIP(ctx.request, ctx.agent.config),
			userIdentifiers: id,
			timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *HTTPRequestRecord) NewUserSignup(id map[string]string) {
	if len(id) == 0 {
		ctx.agent.logger.Warn("TrackSignup(): user id is nil or empty")
		return
	}

	event := &signupUserEvent{
		userEvent: &userEvent{
			ip:              getClientIP(ctx.request, ctx.agent.config),
			userIdentifiers: id,
			timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *HTTPRequestRecord) Close() {
	if !ctx.shouldSend {
		return
	}

	ctx.agent.addRecord(newHTTPRequestRecord(ctx))
}

func (ctx *HTTPRequestRecord) addSilentEvent(event *HTTPRequestEvent) {
	ctx.addEvent_(event, true)
}

func (ctx *HTTPRequestRecord) addEvent(event *HTTPRequestEvent) {
	ctx.addEvent_(event, false)
}

func (ctx *HTTPRequestRecord) addEvent_(event *HTTPRequestEvent, silent bool) {
	ctx.eventsLock.Lock()
	defer ctx.eventsLock.Unlock()
	ctx.events = append(ctx.events, event)
	ctx.shouldSend = !silent
}

func (ctx *HTTPRequestRecord) addUserEvent(event userEventFace) {
	ctx.agent.addUserEvent(event)
}

func (e *HTTPRequestEvent) WithTimestamp(t time.Time) {
	if e == nil {
		return
	}
	e.timestamp = t
}

func (e *HTTPRequestEvent) WithProperties(p map[string]string) {
	if e == nil {
		return
	}
	e.properties = p
}

func (e *HTTPRequestEvent) WithUserIdentifiers(id map[string]string) {
	if e == nil {
		return
	}
	e.userIdentifiers = id
}

func (e *HTTPRequestEvent) GetTime() time.Time {
	return e.timestamp
}

func (e *HTTPRequestEvent) GetName() string {
	return e.method
}

func (e *HTTPRequestEvent) GetEvent() string {
	return e.event
}

func (e *HTTPRequestEvent) GetArgs() (args api.RequestRecord_Observed_SDKEvent_Args) {
	if e.method == sdkMethodTrack {
		args.Args = &api.RequestRecord_Observed_SDKEvent_Args_Track_{api.NewRequestRecord_Observed_SDKEvent_Args_TrackFromFace(e)}
	} else if e.method == sdkMethodIdentify {
		args.Args = &api.RequestRecord_Observed_SDKEvent_Args_Identify_{api.NewRequestRecord_Observed_SDKEvent_Args_IdentifyFromFace(e)}
	}
	return
}

func (e *HTTPRequestEvent) GetOptions() *api.RequestRecord_Observed_SDKEvent_Args_Track_Options {
	return api.NewRequestRecord_Observed_SDKEvent_Args_Track_OptionsFromFace(e)
}

func (e *HTTPRequestEvent) GetProperties() *api.Struct {
	if len(e.properties) == 0 {
		return nil
	}
	return &api.Struct{e.properties}
}

func (e *HTTPRequestEvent) GetUserIdentifiers() *api.Struct {
	if len(e.userIdentifiers) == 0 {
		return nil
	}
	return &api.Struct{e.userIdentifiers}
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
	return getClientIP(r.ctx.request, r.ctx.agent.config)
}

type getClientIPConfigFace interface {
	HTTPClientIPHeader() string
	HTTPClientIPHeaderFormat() string
}

func getClientIP(req *http.Request, cfg getClientIPConfigFace) string {
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

	if prioritizedHeader := cfg.HTTPClientIPHeader(); prioritizedHeader != "" {
		if value := req.Header.Get(prioritizedHeader); value != "" {
			if fmt := cfg.HTTPClientIPHeaderFormat(); fmt != "" {
				parsed, err := parseClientIPHeaderHeaderValue(fmt, value)
				if err == nil {
					// Parsing ok, keep its returned value.
					value = parsed
				} else {
					// An error occured while parsing the header value, so ignore it.
					value = ""
				}
			}

			if value != "" {
				if ip := check(value); ip != nil {
					return ip.String()
				}
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

func parseClientIPHeaderHeaderValue(format, value string) (string, error) {
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

func (r *httpRequestRecord) GetRequest() api.RequestRecord_Request {
	req := r.ctx.request

	trackedHeaders := config.TrackedHTTPHeaders
	if extraHeader := r.ctx.agent.config.HTTPClientIPHeader(); extraHeader != "" {
		trackedHeaders = append(trackedHeaders, extraHeader)
	}
	headers := make([]api.RequestRecord_Request_Header, 0, len(req.Header))
	for _, header := range trackedHeaders {
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
			r.ctx.agent.logger.Error("could not generate a request id ", err)
			requestId = ""
		}
		requestId = hex.EncodeToString(uuid[:])
	}

	var referer string
	if !r.ctx.agent.config.StripHTTPReferer() {
		referer = req.Referer()
	}

	// FIXME: create it from an interface for compile-time error-checking.
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
		Referer:    referer,
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

func isGlobal(ip net.IP) bool {
	if ipv4 := ip.To4(); ipv4 != nil && config.IPv4PublicNetwork.Contains(ipv4) {
		return false
	}
	return !isPrivate(ip)
}

func isPrivate(ip net.IP) bool {
	var privateNetworks []*net.IPNet
	// We cannot rely on `len(ip)` to know what type of IP address this is.
	// `net.ParseIP()` or `net.IPv4()` can return internal 16-byte representations
	// of an IP address even if it is an IPv4. So the trick is to use `ip.To4()`
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

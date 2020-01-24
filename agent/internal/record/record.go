// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package record

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/agent/internal/actor"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/types"
)

type Agent interface {
	FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error)
	FindActionByUserID(userID map[string]string) (action actor.Action, exists bool)
	AddHTTPRequestRecord(rr *RequestRecord)
	AddUserEvent(event UserEventFace)
	IsIPWhitelisted(ip net.IP) (whitelisted bool, matchedCIDR string, err error)
	AddWhitelistEvent(matchedWhitelistEntry string)
}

type Logger interface {
	plog.ErrorLogger
	plog.InfoLogger
}

// RequestRecordForAgentFace is the internal request record interface for the
// agent.
type RequestRecordForAgentFace interface {
	types.RequestRecord
	ClientIP() net.IP
	Request() *http.Request
	Events() []*HTTPRequestEvent
	Attacks() []*AttackEvent
}

// RequestRecord is the internal request record interface.
type RequestRecordFace interface {
	types.RequestRecord
	ClientIP() net.IP
	Request() *http.Request
	SetRequest(*http.Request)
	AddAttackEvent(attack *AttackEvent)
}

type RequestRecord struct {
	// Copy of the request, safe to be asynchronously read, even after the request
	// was terminated.
	request      *http.Request
	eventsLock   sync.Mutex
	events       []*HTTPRequestEvent
	attacks      []*AttackEvent
	identifyOnce sync.Once
	// User-identifiers globally associated to this request using `Identify()`
	userID map[string]string
	// The last non-nil security response is cached and returned to every
	// subsequent calls to method `SecurityResponse()`.
	lastSecurityResponseHandler http.Handler
	// The last non-nil user security response is cached and returned to every
	// subsequent calls to method `UserSecurityResponse()`. Middleware functions
	// can therefore observe the same result with `UserSecurityResponse()` as in
	// the request handler with `MatchSecurityResponse()`.
	lastUserSecurityResponseHandler http.Handler
	agent                           Agent
	logger                          Logger
	shouldSend                      bool
	// clientIP value deduced from the request headers.
	clientIP net.IP
}

func (rr *RequestRecord) SetRequest(r *http.Request) {
	rr.request = r
}

func (rr *RequestRecord) AddAttackEvent(attack *AttackEvent) {
	rr.eventsLock.Lock()
	defer rr.eventsLock.Unlock()
	rr.attacks = append(rr.attacks, attack)
	rr.shouldSend = true
}

func (rr *RequestRecord) ClientIP() net.IP {
	return rr.clientIP
}

func (rr *RequestRecord) Request() *http.Request {
	return rr.request
}

func (rr *RequestRecord) Events() []*HTTPRequestEvent {
	return rr.events
}

func (rr *RequestRecord) Attacks() []*AttackEvent {
	return rr.attacks
}

type RequestRecordContextKey struct{}

func NewRequestRecord(agent Agent, logger Logger, req *http.Request, cfg getClientIPConfigFace) (RequestRecordFace, *http.Request) {
	clientIP := getClientIP(req, cfg)
	whitelisted, matched, err := agent.IsIPWhitelisted(clientIP)
	if err != nil {
		logger.Error(err)
		whitelisted = false
	}
	var rr RequestRecordFace
	if whitelisted {
		agent.AddWhitelistEvent(matched)
		rr = WhitelistedHTTPRequestRecord{
			clientIP: clientIP,
		}

	} else {
		rr = &RequestRecord{
			agent:    agent,
			logger:   logger,
			clientIP: clientIP,
		}
	}

	// Store the internal request record into the request context too in order to
	// get it without going through the SDK - we don't want users to be able
	// to access the internal request record from the SDK API.
	ctx := req.Context()
	req = req.WithContext(context.WithValue(ctx, RequestRecordContextKey{}, rr))

	// Set the record request pointer the one that was just created by
	// `req.WithContext()` which has copied the original request to modify its
	// context.
	rr.SetRequest(req)

	return rr, req
}

func FromContext(ctx context.Context) RequestRecordFace {
	return ctx.Value(RequestRecordContextKey{}).(RequestRecordFace)
}

func IsWhitelisted(r *http.Request) bool {
	_, ok := FromContext(r.Context()).(WhitelistedHTTPRequestRecord)
	return ok
}

type HTTPRequestEvent struct {
	method          string
	event           string
	properties      types.EventProperties
	userIdentifiers EventUserIdentifiersMap
	timestamp       time.Time
}

type AttackEvent struct {
	Rule      string
	Test      bool
	Blocked   bool
	Timestamp time.Time
	Info      interface{}
}

type UserEventFace interface {
	isUserEvent()
}

type UserEvent struct {
	UserIdentifiers EventUserIdentifiersMap
	Timestamp       time.Time
	IP              net.IP
}

type AuthUserEvent struct {
	*UserEvent
	LoginSuccess bool
}

func (_ *AuthUserEvent) isUserEvent() {}

type SignupUserEvent struct {
	*UserEvent
}

func (_ *SignupUserEvent) isUserEvent() {}

type EventPropertyMap map[string]string

type EventUserIdentifiersMap map[string]string

const (
	sdkMethodIdentify = "identify"
	sdkMethodTrack    = "track"
)

func (ctx *RequestRecord) NewCustomEvent(event string) types.CustomEvent {
	evt := &HTTPRequestEvent{
		method:    sdkMethodTrack,
		event:     event,
		timestamp: time.Now(),
	}
	ctx.addEvent(evt)
	return evt
}

func (ctx *RequestRecord) Identify(id map[string]string) {
	ctx.identifyOnce.Do(func() {
		evt := &HTTPRequestEvent{
			method:          sdkMethodIdentify,
			timestamp:       time.Now(),
			userIdentifiers: id,
		}
		// Globally associate these user-identifiers with the request.
		ctx.userID = id
		ctx.addSilentEvent(evt)
	})
}

func (ctx *RequestRecord) SecurityResponse() http.Handler {
	if ctx.lastSecurityResponseHandler != nil {
		return ctx.lastSecurityResponseHandler
	}
	agent := ctx.agent
	ip := ctx.clientIP
	action, exists, err := agent.FindActionByIP(ip)
	if err != nil {
		ctx.logger.Error(err)
		return nil
	}
	if !exists {
		return nil
	}
	ctx.lastSecurityResponseHandler, err = actor.NewIPActionHTTPHandler(action, ip)
	if err != nil {
		ctx.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("could not create the http handler for an IP security response: action `%v` - IP `%s`:", action.ActionID(), ip)))
	}
	return ctx.lastSecurityResponseHandler
}

func (ctx *RequestRecord) UserSecurityResponse() http.Handler {
	userID := ctx.userID
	if userID == nil {
		return nil
	}
	if ctx.lastUserSecurityResponseHandler != nil {
		return ctx.lastUserSecurityResponseHandler
	}
	agent := ctx.agent
	action, exists := agent.FindActionByUserID(userID)
	if !exists {
		return nil
	}
	var err error
	ctx.lastUserSecurityResponseHandler, err = actor.NewUserActionHTTPHandler(action, userID)
	if err != nil {
		ctx.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("could not create the http handler for a user security response: action `%v` - user `%v`:", action.ActionID(), userID)))
	}
	return ctx.lastUserSecurityResponseHandler
}

func (ctx *RequestRecord) NewUserAuth(id map[string]string, loginSuccess bool) {
	if len(id) == 0 {
		ctx.logger.Info("TrackAuth(): user id is nil or empty")
		return
	}

	event := &AuthUserEvent{
		LoginSuccess: loginSuccess,
		UserEvent: &UserEvent{
			IP:              ctx.clientIP,
			UserIdentifiers: id,
			Timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *RequestRecord) NewUserSignup(id map[string]string) {
	if len(id) == 0 {
		ctx.logger.Info("TrackSignup(): user id is nil or empty")
		return
	}

	event := &SignupUserEvent{
		UserEvent: &UserEvent{
			IP:              ctx.clientIP,
			UserIdentifiers: id,
			Timestamp:       time.Now(),
		},
	}
	ctx.addUserEvent(event)
}

func (ctx *RequestRecord) Close() {
	if !ctx.shouldSend {
		return
	}

	ctx.agent.AddHTTPRequestRecord(ctx)
}

func (ctx *RequestRecord) Whitelisted() bool {
	return false
}

func (ctx *RequestRecord) addSilentEvent(event *HTTPRequestEvent) {
	ctx.addEvent_(event, true)
}

func (ctx *RequestRecord) addEvent(event *HTTPRequestEvent) {
	ctx.addEvent_(event, false)
}

func (ctx *RequestRecord) addEvent_(event *HTTPRequestEvent, silent bool) {
	ctx.eventsLock.Lock()
	defer ctx.eventsLock.Unlock()
	ctx.events = append(ctx.events, event)
	ctx.shouldSend = !silent
}

func (ctx *RequestRecord) addUserEvent(event UserEventFace) {
	// User events don't go through the request record event list but through
	// aggregated metrics.
	ctx.agent.AddUserEvent(event)
}

func (e *HTTPRequestEvent) WithTimestamp(t time.Time) {
	if e == nil {
		return
	}
	e.timestamp = t
}

func (e *HTTPRequestEvent) WithProperties(p types.EventProperties) {
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
	if e.properties == nil {
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

// WhitelistedHTTPRequestRecord is a request record whose methods do nothing in
// order to whitelist the request.
type WhitelistedHTTPRequestRecord struct {
	clientIP net.IP
	request  *http.Request
}

func (rr WhitelistedHTTPRequestRecord) SetRequest(r *http.Request)              { rr.request = r }
func (WhitelistedHTTPRequestRecord) NewUserSignup(map[string]string)            {}
func (WhitelistedHTTPRequestRecord) NewUserAuth(map[string]string, bool)        {}
func (WhitelistedHTTPRequestRecord) Identify(map[string]string)                 {}
func (WhitelistedHTTPRequestRecord) SecurityResponse() http.Handler             { return nil }
func (WhitelistedHTTPRequestRecord) UserSecurityResponse() http.Handler         { return nil }
func (rr WhitelistedHTTPRequestRecord) NewCustomEvent(string) types.CustomEvent { return rr }
func (WhitelistedHTTPRequestRecord) Close()                                     {}
func (WhitelistedHTTPRequestRecord) WithTimestamp(time.Time)                    {}
func (WhitelistedHTTPRequestRecord) WithProperties(types.EventProperties)       {}
func (WhitelistedHTTPRequestRecord) WithUserIdentifiers(map[string]string)      {}
func (WhitelistedHTTPRequestRecord) Whitelisted() bool                          { return true }
func (rr WhitelistedHTTPRequestRecord) ClientIP() net.IP                        { return rr.clientIP }
func (rr WhitelistedHTTPRequestRecord) Request() *http.Request                  { return rr.request }
func (WhitelistedHTTPRequestRecord) Events() []*HTTPRequestEvent                { return nil }
func (WhitelistedHTTPRequestRecord) AddAttackEvent(*AttackEvent)                {}

type getClientIPConfigFace interface {
	HTTPClientIPHeader() string
	HTTPClientIPHeaderFormat() string
}

func getClientIP(req *http.Request, cfg getClientIPConfigFace) net.IP {
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
					return ip
				}
			}
		}
	}

	for _, key := range config.IPRelatedHTTPHeaders {
		value := req.Header.Get(key)
		if ip := check(value); ip != nil {
			return ip
		}
	}

	remoteIPStr, _ := SplitHostPort(req.RemoteAddr)
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
func SplitHostPort(addr string) (host string, port string) {
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

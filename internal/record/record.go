// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package record

//type Agent interface {
//	FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error)
//	FindActionByUserID(userID map[string]string) (action actor.Action, exists bool)
//	AddHTTPRequestRecord(rr *RequestRecord)
//	addUserEvent(event UserEventFace)
//	IsIPWhitelisted(ip net.IP) (whitelisted bool, matchedCIDR string, err error)
//	addWhitelistEvent(matchedWhitelistEntry string)
//}
//
//type Logger interface {
//	plog.ErrorLogger
//	plog.InfoLogger
//}

// RequestRecordForAgentFace is the internal request record interface for the
// agent.
//type RequestRecordForAgentFace interface {
//	types.RequestRecord
//	ClientIP() net.IP
//	Request() *http.Request
//	Events() []*HTTPRequestEvent
//	Attacks() []*AttackEvent
//}
//
//// RequestRecord is the internal request record interface.
//type RequestRecordFace interface {
//	types.RequestRecord
//	ClientIP() net.IP
//	Request() *http.Request
//	SetRequest(*http.Request)
//	AddAttackEvent(attack *AttackEvent)
//}
//
//type RequestRecordContextKey struct{}
//
//func FromContext(ctx context.Context) RequestRecordFace {
//	return ctx.Value(RequestRecordContextKey{}).(RequestRecordFace)
//}

//
//type AttackEventAPIAdaptor record.AttackEvent
//
//type RequestAPIAdaptor struct {
//	*HTTPRequestRecordEvent
//	cache struct {
//		remoteIP, remotePort, hostPort string
//	}
//}
//
//func (a *RequestAPIAdaptor) request() *http.Request {
//	return a.rr.Request()
//}

// WhitelistedHTTPRequestRecord is a request record whose methods do nothing in
// order to whitelist the request.
//type WhitelistedHTTPRequestRecord struct {
//	clientIP net.IP
//	request  *http.Request
//}
//
//func (rr WhitelistedHTTPRequestRecord) SetRequest(r *http.Request)              { rr.request = r }
//func (WhitelistedHTTPRequestRecord) AddUserSignup(map[string]string)            {}
//func (WhitelistedHTTPRequestRecord) NewUserAuth(map[string]string, bool)        {}
//func (WhitelistedHTTPRequestRecord) Identify(map[string]string)                 {}
//func (WhitelistedHTTPRequestRecord) SecurityResponse() http.Handler             { return nil }
//func (WhitelistedHTTPRequestRecord) UserSecurityResponse() http.Handler         { return nil }
//func (rr WhitelistedHTTPRequestRecord) AddCustomEvent(string) types.CustomEvent { return rr }
//func (WhitelistedHTTPRequestRecord) Close()                                     {}
//func (WhitelistedHTTPRequestRecord) WithTimestamp(time.Time)                    {}
//func (WhitelistedHTTPRequestRecord) WithProperties(types.EventProperties)       {}
//func (WhitelistedHTTPRequestRecord) WithUserIdentifiers(map[string]string)      {}
//func (WhitelistedHTTPRequestRecord) Whitelisted() bool                          { return true }
//func (rr WhitelistedHTTPRequestRecord) ClientIP() net.IP                        { return rr.clientIP }
//func (rr WhitelistedHTTPRequestRecord) Request() *http.Request                  { return rr.request }
//func (WhitelistedHTTPRequestRecord) Events() []*HTTPRequestEvent                { return nil }
//func (WhitelistedHTTPRequestRecord) AddAttackEvent(*AttackEvent)                {}
//
//type getClientIPConfigFace interface {
//	HTTPClientIPHeader() string
//	HTTPClientIPHeaderFormat() string
//}

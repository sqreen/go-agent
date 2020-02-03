// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api

import (
	"encoding/json"
	"time"

	"github.com/sqreen/go-agent/internal/sqlib/sqsanitize"
)

type AppLoginRequest struct {
	BundleSignature  string                       `protobuf:"bytes,1,opt,name=bundle_signature,json=bundleSignature,proto3" json:"bundle_signature"`
	VariousInfos     AppLoginRequest_VariousInfos `protobuf:"bytes,2,opt,name=various_infos,json=variousInfos,proto3" json:"various_infos"`
	AgentType        string                       `protobuf:"bytes,3,opt,name=agent_type,json=agentType,proto3" json:"agent_type"`
	AgentVersion     string                       `protobuf:"bytes,4,opt,name=agent_version,json=agentVersion,proto3" json:"agent_version"`
	OsType           string                       `protobuf:"bytes,5,opt,name=os_type,json=osType,proto3" json:"os_type"`
	Hostname         string                       `protobuf:"bytes,6,opt,name=hostname,proto3" json:"hostname"`
	RuntimeType      string                       `protobuf:"bytes,7,opt,name=runtime_type,json=runtimeType,proto3" json:"runtime_type"`
	RuntimeVersion   string                       `protobuf:"bytes,8,opt,name=runtime_version,json=runtimeVersion,proto3" json:"runtime_version"`
	FrameworkType    string                       `protobuf:"bytes,9,opt,name=framework_type,json=frameworkType,proto3" json:"framework_type"`
	FrameworkVersion string                       `protobuf:"bytes,10,opt,name=framework_version,json=frameworkVersion,proto3" json:"framework_version"`
	Environment      string                       `protobuf:"bytes,11,opt,name=environment,proto3" json:"environment"`
}

type AppLoginRequest_VariousInfos struct {
	Time             time.Time `protobuf:"bytes,1,opt,name=time,proto3,stdtime" json:"time"`
	Pid              uint32    `protobuf:"varint,3,opt,name=pid,proto3" json:"pid"`
	Ppid             uint32    `protobuf:"varint,4,opt,name=ppid,proto3" json:"ppid"`
	Euid             uint32    `protobuf:"varint,5,opt,name=euid,proto3" json:"euid"`
	Egid             uint32    `protobuf:"varint,6,opt,name=egid,proto3" json:"egid"`
	Uid              uint32    `protobuf:"varint,7,opt,name=uid,proto3" json:"uid"`
	Gid              uint32    `protobuf:"varint,8,opt,name=gid,proto3" json:"gid"`
	Name             string    `protobuf:"bytes,9,opt,name=name,proto3" json:"name"`
	LibSqreenVersion *string   `json:"libsqreen_version"`
}

type AppLoginResponse struct {
	Error     string                   `json:"error"`
	SessionId string                   `protobuf:"bytes,1,opt,name=session_id,json=sessionId,proto3" json:"session_id"`
	Status    bool                     `protobuf:"varint,2,opt,name=status,proto3" json:"status"`
	Commands  []CommandRequest         `protobuf:"bytes,3,rep,name=commands,proto3" json:"commands"`
	Features  AppLoginResponse_Feature `protobuf:"bytes,4,opt,name=features,proto3" json:"features"`
	PackId    string                   `protobuf:"bytes,5,opt,name=pack_id,json=packId,proto3" json:"pack_id"`
}

type AppLoginResponse_Feature struct {
	BatchSize      uint32 `protobuf:"varint,1,opt,name=batch_size,json=batchSize,proto3" json:"batch_size"`
	MaxStaleness   uint32 `protobuf:"varint,2,opt,name=max_staleness,json=maxStaleness,proto3" json:"max_staleness"`
	HeartbeatDelay uint32 `protobuf:"varint,3,opt,name=heartbeat_delay,json=heartbeatDelay,proto3" json:"heartbeat_delay"`
}

type CommandRequest struct {
	Name string `json:"name"`
	Uuid string `json:"uuid"`
	// Params: parse and validate the params when used.
	Params []json.RawMessage `json:"params"`
}

type CommandResult struct {
	Output string `protobuf:"bytes,1,opt,name=output,proto3" json:"output"`
	Status bool   `protobuf:"varint,2,opt,name=status,proto3" json:"status"`
}

type MetricResponse struct {
	Name        string    `protobuf:"bytes,1,opt,name=name,proto3" json:"name"`
	Start       time.Time `protobuf:"bytes,2,opt,name=start,proto3,stdtime" json:"start"`
	Finish      time.Time `protobuf:"bytes,3,opt,name=finish,proto3,stdtime" json:"finish"`
	Observation Struct    `protobuf:"bytes,4,opt,name=observation,proto3,customtype=Struct" json:"observation"`
}

type AppBeatRequest struct {
	CommandResults map[string]CommandResult `json:"command_results,omitempty"`
	Metrics        []MetricResponse         `json:"metrics,omitempty"`
}

type AppBeatResponse struct {
	Commands []CommandRequest `json:"commands,omitempty"`
	Status   bool             `json:"status"`
}

type BatchRequest struct {
	Batch []BatchRequest_Event `json:"batch"`
}

type RequestRecordEvent struct{ *RequestRecord }

func (RequestRecordEvent) GetEventType() string {
	return "request_record"
}

func (e RequestRecordEvent) GetEvent() Struct {
	return Struct{e}
}

type ExceptionEvent struct {
	Time        time.Time        `json:"time"`
	Klass       string           `json:"klass"`
	Message     string           `json:"message"`
	RulespackID string           `json:"rulespack_id"`
	Context     ExceptionContext `json:"context"`
	Infos       interface{}      `json:"infos,omitempty"`
}

type ExceptionEventFace interface {
	GetTime() time.Time
	GetKlass() string
	GetMessage() string
	GetRulespackID() string
	GetContext() ExceptionContext
	GetInfos() interface{}
}

func NewExceptionEventFromFace(e ExceptionEventFace) *ExceptionEvent {
	return &ExceptionEvent{
		Time:        e.GetTime(),
		Klass:       e.GetKlass(),
		Message:     e.GetMessage(),
		Context:     e.GetContext(),
		RulespackID: e.GetRulespackID(),
		Infos:       e.GetInfos(),
	}
}

func (*ExceptionEvent) GetEventType() string {
	return "sqreen_exception"
}

func (e *ExceptionEvent) GetEvent() Struct {
	return Struct{e}
}

type ExceptionContext struct {
	Backtrace []StackFrame `json:"backtrace,omitempty"`
}

type ExceptionContextFace interface {
	GetBacktrace() []StackFrame
}

func NewExceptionContextFromFace(c ExceptionContextFace) *ExceptionContext {
	return &ExceptionContext{Backtrace: c.GetBacktrace()}
}

type StackFrame struct {
	Method     string `json:"method"`
	File       string `json:"file"`
	LineNumber uint32 `json:"line_number"`
}

type StackFrameFace interface {
	GetMethod() string
	GetFile() string
	GetLineNumber() uint32
}

func NewStackFrameFromFace(e StackFrameFace) *StackFrame {
	return &StackFrame{
		Method:     e.GetMethod(),
		File:       e.GetFile(),
		LineNumber: e.GetLineNumber()}
}

type BatchRequest_Event struct {
	EventType string `protobuf:"bytes,1,opt,name=event_type,json=eventType,proto3" json:"event_type"`
	Event     Struct `protobuf:"bytes,2,opt,name=event,proto3,customtype=Struct" json:"event"`
}

type Rule struct {
	Name       string             `json:"name"`
	Hookpoint  Hookpoint          `json:"hookpoint"`
	Data       RuleData           `json:"data"`
	Metrics    []MetricDefinition `json:"metrics"`
	Signature  RuleSignature      `json:"signature"`
	Conditions RuleConditions     `json:"conditions"`
	Callbacks  RuleCallbacks      `json:"callbacks"`
	Test       bool               `json:"test"`
	Block      bool               `json:"block"`
}

type RuleConditions struct{}
type RuleCallbacks struct{}

type ECDSASignature struct {
	Keys  []string `json:"keys"`
	Value string   `json:"value"`
	// Custom field where the signed message is reconstructed out of the list of
	// keys
	Message []byte `json:"-"`
}

type RuleSignature struct {
	ECDSASignature ECDSASignature `json:"v0_9"`
}

type MetricDefinition struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Period int64  `json:"period"`
}

type Hookpoint struct {
	Class    string `json:"klass"`
	Method   string `json:"method"`
	Callback string `json:"callback_class"`
}

type RuleData struct {
	Values []RuleDataEntry `json:"values"`
}

type RuleDataEntry Struct

const (
	CustomErrorPageType = "custom_error_page"
	RedirectionType     = "redirection"
	WAFType             = "waf"
)

type CustomErrorPageRuleDataEntry struct {
	StatusCode int `json:"status_code"`
}

type RedirectionRuleDataEntry struct {
	RedirectionURL string `json:"redirection_url"`
}

type WAFRuleDataEntry struct {
	BindingAccessors []string `json:"binding_accessors"`
	WAFRules         string   `json:"waf_rules"`
	Timeout          uint64   `json:"max_budget_ms"`
}

type Dependency struct {
	Name     string             `protobuf:"bytes,1,opt,name=name,proto3" json:"name"`
	Version  string             `protobuf:"bytes,2,opt,name=version,proto3" json:"version"`
	Homepage string             `protobuf:"bytes,3,opt,name=homepage,proto3" json:"homepage"`
	Source   *Dependency_Source `protobuf:"bytes,4,opt,name=source,proto3" json:"source"`
}

type Dependency_Source struct {
	Name    string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name"`
	Remotes []string `protobuf:"bytes,2,rep,name=remotes,proto3" json:"remotes"`
}

type RequestRecord struct {
	Version     string                 `protobuf:"bytes,1,opt,name=version,proto3" json:"version"`
	RulespackId string                 `protobuf:"bytes,2,opt,name=rulespack_id,json=rulespackId,proto3" json:"rulespack_id"`
	ClientIp    string                 `protobuf:"bytes,3,opt,name=client_ip,json=clientIp,proto3" json:"client_ip"`
	Request     RequestRecord_Request  `protobuf:"bytes,4,opt,name=request,proto3" json:"request"`
	Response    RequestRecord_Response `protobuf:"bytes,5,opt,name=response,proto3" json:"response"`
	Observed    RequestRecord_Observed `protobuf:"bytes,6,opt,name=observed,proto3" json:"observed"`
}

func (rr *RequestRecord) Scrub(scrubber *sqsanitize.Scrubber, info sqsanitize.Info) (scrubbed bool, err error) {
	var (
		scrubbedRequest, scrubbedObserved bool
		requestScrubbingInfo              = sqsanitize.Info{}
	)

	// Temporary hack to scrub the WAF data that copied some request data.
	// firstly, scrub the request and pass its scrubbing information to the
	// scrubbing of the observations which may include the WAF information.
	// The WAFAttackInfo type implements the CustomScrubber interface and expects
	// this value.
	scrubbedRequest, err = scrubber.Scrub(&rr.Request, requestScrubbingInfo)
	if err != nil {
		return
	}
	scrubbedObserved, err = scrubber.Scrub(&rr.Observed, requestScrubbingInfo)
	if err != nil {
		return
	}
	info.Append(requestScrubbingInfo)
	return scrubbedRequest || scrubbedObserved, nil
}

type RequestRecord_Request struct {
	Rid        string                           `json:"rid"`
	Headers    []RequestRecord_Request_Header   `json:"headers"`
	Verb       string                           `json:"verb"`
	Path       string                           `json:"path"`
	RawPath    string                           `json:"raw_path"`
	Host       string                           `json:"host"`
	Port       string                           `json:"port"`
	RemoteIp   string                           `json:"remote_ip"`
	RemotePort string                           `json:"remote_port"`
	Scheme     string                           `json:"scheme"`
	UserAgent  string                           `json:"user_agent"`
	Referer    string                           `json:"referer"`
	Parameters RequestRecord_Request_Parameters `json:"parameters"`
}

type RequestRecord_Request_Header struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value"`
}

type RequestRecord_Request_Parameters struct {
	// Query parameters
	Query map[string][]string `json:"query,omitempty"`
	// application/x-www-form-urlencoded or multipart/form-data parameters
	Form map[string][]string `json:"form,omitempty"`
}

type RequestRecord_Response struct {
	Status        uint32 `protobuf:"varint,1,opt,name=status,proto3" json:"status"`
	ContentLength uint32 `protobuf:"varint,2,opt,name=content_length,json=contentLength,proto3" json:"content_length"`
	ContentType   string `protobuf:"bytes,3,opt,name=content_type,json=contentType,proto3" json:"content_type"`
}

type RequestRecord_Observed struct {
	Attacks          []*RequestRecord_Observed_Attack      `protobuf:"bytes,1,rep,name=attacks,proto3" json:"attacks,omitempty"`
	Sdk              []*RequestRecord_Observed_SDKEvent    `protobuf:"bytes,2,rep,name=sdk,proto3" json:"sdk,omitempty"`
	SqreenExceptions []*RequestRecord_Observed_Exception   `protobuf:"bytes,3,rep,name=sqreen_exceptions,json=sqreenExceptions,proto3" json:"sqreen_exceptions,omitempty"`
	Observations     []*RequestRecord_Observed_Observation `protobuf:"bytes,4,rep,name=observations,proto3" json:"observations,omitempty"`
	DataPoints       []*RequestRecord_Observed_DataPoint   `protobuf:"bytes,5,rep,name=data_points,json=dataPoints,proto3" json:"data_points,omitempty"`
}

type RequestRecord_Observed_Attack struct {
	RuleName string      `protobuf:"bytes,1,opt,name=rule_name,json=ruleName,proto3" json:"rule_name"`
	Test     bool        `protobuf:"varint,2,opt,name=test,proto3" json:"test"`
	Info     interface{} `protobuf:"bytes,3,opt,name=infos,proto3" json:"infos"`
	Time     time.Time   `protobuf:"bytes,5,opt,name=time,proto3,stdtime" json:"time"`
	Block    bool        `protobuf:"varint,6,opt,name=block,proto3" json:"block"`
}

type WAFAttackInfo struct {
	WAFData string `json:"waf_data"`
}

type WAFInfoFilter struct {
	Operator        string `json:"operator"`
	OperatorValue   string `json:"operator_value"`
	BindingAccessor string `json:"binding_accessor"`
	ResolvedValue   string `json:"resolved_value"`
	MatchStatus     string `json:"match_status,omitempty"`
}
type WAFInfo struct {
	RetCode int             `json:"ret_code"`
	Flow    string          `json:"flow"`
	Step    string          `json:"step"`
	Rule    string          `json:"rule"`
	Filter  []WAFInfoFilter `json:"filter"`
}

// Scrub of WAF attack information by implementing spec 22. This is a temporary
// solution until a better WAF API is provided that would avoid that.
func (i *WAFAttackInfo) Scrub(scrubber *sqsanitize.Scrubber, info sqsanitize.Info) (scrubbed bool, err error) {
	if len(info) == 0 {
		return false, nil
	}

	// Unmarshal the WAF attack information that was returned by the WAF.
	var wafInfo []WAFInfo
	if err := json.Unmarshal([]byte(i.WAFData), &wafInfo); err != nil {
		return false, err
	}

	// Walk the WAF information and redact resolved binding accessor values that
	// were scrubbed. The caller must have stored into info the values scrubbed
	// from the request.
	redactedString := scrubber.RedactedValueMask()
	for e := range wafInfo {
		for f := range wafInfo[e].Filter {
			if info.Contains(wafInfo[e].Filter[f].ResolvedValue) {
				wafInfo[e].Filter[f].ResolvedValue = redactedString
				if wafInfo[e].Filter[f].MatchStatus != "" {
					wafInfo[e].Filter[f].MatchStatus = redactedString
				}
				scrubbed = true
			}
		}
	}

	if !scrubbed {
		return false, nil
	}

	// Marshal back to json the scrubbed WAF info
	buf, err := json.Marshal(&wafInfo)
	if err != nil {
		return false, err
	}
	i.WAFData = string(buf)
	return scrubbed, nil
}

type RequestRecord_Observed_SDKEvent struct {
	Time time.Time                            `protobuf:"bytes,1,opt,name=time,proto3,stdtime" json:"time"`
	Name string                               `protobuf:"bytes,2,opt,name=name,proto3" json:"name"`
	Args RequestRecord_Observed_SDKEvent_Args `protobuf:"bytes,3,opt,name=args,proto3" json:"args"`
}

// Helper message type to disable the face extension only on it and not in
// the entire SDKEvent message type. oneof + face is not supported.
type RequestRecord_Observed_SDKEvent_Args struct {
	// Types that are valid to be assigned to Args:
	//	*RequestRecord_Observed_SDKEvent_Args_Track_
	//	*RequestRecord_Observed_SDKEvent_Args_Identify_
	Args isRequestRecord_Observed_SDKEvent_Args_Args `protobuf_oneof:"args"`
}

type isRequestRecord_Observed_SDKEvent_Args_Args interface {
	isRequestRecord_Observed_SDKEvent_Args_Args()
}

type RequestRecord_Observed_SDKEvent_Args_Track_ struct {
	Track *RequestRecord_Observed_SDKEvent_Args_Track `protobuf:"bytes,1,opt,name=track,proto3,oneof"`
}
type RequestRecord_Observed_SDKEvent_Args_Identify_ struct {
	Identify *RequestRecord_Observed_SDKEvent_Args_Identify `protobuf:"bytes,2,opt,name=identify,proto3,oneof"`
}

func (*RequestRecord_Observed_SDKEvent_Args_Track_) isRequestRecord_Observed_SDKEvent_Args_Args()    {}
func (*RequestRecord_Observed_SDKEvent_Args_Identify_) isRequestRecord_Observed_SDKEvent_Args_Args() {}

// Serialized into:
// [
//   "<name>",
//   {
//     "user_identifiers": <user_identifiers>,
//     "properties": <properties>
//   }
// ]
type RequestRecord_Observed_SDKEvent_Args_Track struct {
	Event   string                                              `protobuf:"bytes,1,opt,name=event,proto3" json:"event"`
	Options *RequestRecord_Observed_SDKEvent_Args_Track_Options `protobuf:"bytes,2,opt,name=options,proto3" json:"options"`
}

type RequestRecord_Observed_SDKEvent_Args_Track_Options struct {
	Properties      *Struct `protobuf:"bytes,1,opt,name=properties,proto3,customtype=Struct" json:"properties,omitempty"`
	UserIdentifiers *Struct `protobuf:"bytes,2,opt,name=user_identifiers,json=userIdentifiers,proto3,customtype=Struct" json:"user_identifiers,omitempty"`
}

// Serialized into:
// [ <user_identifiers> ]
type RequestRecord_Observed_SDKEvent_Args_Identify struct {
	UserIdentifiers *Struct `protobuf:"bytes,1,opt,name=user_identifiers,json=userIdentifiers,proto3,customtype=Struct" json:"user_identifiers"`
}

type RequestRecord_Observed_Exception struct {
	Message   string    `protobuf:"bytes,1,opt,name=message,proto3" json:"message"`
	Klass     string    `protobuf:"bytes,2,opt,name=klass,proto3" json:"klass"`
	RuleName  string    `protobuf:"bytes,3,opt,name=rule_name,json=ruleName,proto3" json:"rule_name"`
	Test      bool      `protobuf:"varint,4,opt,name=test,proto3" json:"test"`
	Infos     string    `protobuf:"bytes,5,opt,name=infos,proto3" json:"infos"`
	Backtrace []string  `protobuf:"bytes,6,rep,name=backtrace,proto3" json:"backtrace"`
	Time      time.Time `protobuf:"bytes,7,opt,name=time,proto3,stdtime" json:"time"`
}

type RequestRecord_Observed_Observation struct {
	Category string    `protobuf:"bytes,1,opt,name=category,proto3" json:"category"`
	Key      string    `protobuf:"bytes,2,opt,name=key,proto3" json:"key"`
	Value    string    `protobuf:"bytes,3,opt,name=value,proto3" json:"value"`
	Time     time.Time `protobuf:"bytes,4,opt,name=time,proto3,stdtime" json:"time"`
}

type RequestRecord_Observed_DataPoint struct {
	RulespackId string    `protobuf:"bytes,1,opt,name=rulespack_id,json=rulespackId,proto3" json:"rulespack_id"`
	RuleName    string    `protobuf:"bytes,2,opt,name=rule_name,json=ruleName,proto3" json:"rule_name"`
	Time        time.Time `protobuf:"bytes,3,opt,name=time,proto3,stdtime" json:"time"`
	Infos       string    `protobuf:"bytes,4,opt,name=infos,proto3" json:"infos"`
}

type AppLoginRequestFace interface {
	GetBundleSignature() string
	GetVariousInfos() AppLoginRequest_VariousInfos
	GetAgentType() string
	GetAgentVersion() string
	GetOsType() string
	GetHostname() string
	GetRuntimeType() string
	GetRuntimeVersion() string
	GetFrameworkType() string
	GetFrameworkVersion() string
	GetEnvironment() string
}

func NewAppLoginRequestFromFace(that AppLoginRequestFace) *AppLoginRequest {
	this := &AppLoginRequest{}
	this.BundleSignature = that.GetBundleSignature()
	this.VariousInfos = that.GetVariousInfos()
	this.AgentType = that.GetAgentType()
	this.AgentVersion = that.GetAgentVersion()
	this.OsType = that.GetOsType()
	this.Hostname = that.GetHostname()
	this.RuntimeType = that.GetRuntimeType()
	this.RuntimeVersion = that.GetRuntimeVersion()
	this.FrameworkType = that.GetFrameworkType()
	this.FrameworkVersion = that.GetFrameworkVersion()
	this.Environment = that.GetEnvironment()
	return this
}

type AppLoginRequest_VariousInfosFace interface {
	GetTime() time.Time
	GetPid() uint32
	GetPpid() uint32
	GetEuid() uint32
	GetEgid() uint32
	GetUid() uint32
	GetGid() uint32
	GetName() string
	GetLibSqreenVersion() *string
}

func NewAppLoginRequest_VariousInfosFromFace(that AppLoginRequest_VariousInfosFace) *AppLoginRequest_VariousInfos {
	this := &AppLoginRequest_VariousInfos{}
	this.Time = that.GetTime()
	this.Pid = that.GetPid()
	this.Ppid = that.GetPpid()
	this.Euid = that.GetEuid()
	this.Egid = that.GetEgid()
	this.Uid = that.GetUid()
	this.Gid = that.GetGid()
	this.Name = that.GetName()
	this.LibSqreenVersion = that.GetLibSqreenVersion()
	return this
}

type AppLoginResponseFace interface {
	GetSessionId() string
	GetStatus() bool
	GetCommands() []CommandRequest
	GetFeatures() AppLoginResponse_Feature
	GetPackId() string
}

func NewAppLoginResponseFromFace(that AppLoginResponseFace) *AppLoginResponse {
	this := &AppLoginResponse{}
	this.SessionId = that.GetSessionId()
	this.Status = that.GetStatus()
	this.Commands = that.GetCommands()
	this.Features = that.GetFeatures()
	this.PackId = that.GetPackId()
	return this
}

type AppLoginResponse_FeatureFace interface {
	GetBatchSize() uint32
	GetMaxStaleness() uint32
	GetHeartbeatDelay() uint32
}

func NewAppLoginResponse_FeatureFromFace(that AppLoginResponse_FeatureFace) *AppLoginResponse_Feature {
	this := &AppLoginResponse_Feature{}
	this.BatchSize = that.GetBatchSize()
	this.MaxStaleness = that.GetMaxStaleness()
	this.HeartbeatDelay = that.GetHeartbeatDelay()
	return this
}

type CommandResultFace interface {
	GetOutput() string
	GetStatus() bool
}

func NewCommandResultFromFace(that CommandResultFace) *CommandResult {
	this := &CommandResult{}
	this.Output = that.GetOutput()
	this.Status = that.GetStatus()
	return this
}

type MetricResponseFace interface {
	GetName() string
	GetStart() time.Time
	GetFinish() time.Time
	GetObservation() Struct
}

func NewMetricResponseFromFace(that MetricResponseFace) *MetricResponse {
	this := &MetricResponse{}
	this.Name = that.GetName()
	this.Start = that.GetStart()
	this.Finish = that.GetFinish()
	this.Observation = that.GetObservation()
	return this
}

type AppBeatRequestFace interface {
	GetCommandResults() map[string]CommandResult
	GetMetrics() []MetricResponse
}

func NewAppBeatRequestFromFace(that AppBeatRequestFace) *AppBeatRequest {
	this := &AppBeatRequest{}
	this.CommandResults = that.GetCommandResults()
	this.Metrics = that.GetMetrics()
	return this
}

type AppBeatResponseFace interface {
	GetCommands() []CommandRequest
	GetStatus() bool
}

func NewAppBeatResponseFromFace(that AppBeatResponseFace) *AppBeatResponse {
	this := &AppBeatResponse{}
	this.Commands = that.GetCommands()
	this.Status = that.GetStatus()
	return this
}

type BatchRequestFace interface {
	GetBatch() []BatchRequest_Event
}

func NewBatchRequestFromFace(that BatchRequestFace) *BatchRequest {
	this := &BatchRequest{}
	this.Batch = that.GetBatch()
	return this
}

type BatchRequest_EventFace interface {
	GetEventType() string
	GetEvent() Struct
}

func NewBatchRequest_EventFromFace(that BatchRequest_EventFace) *BatchRequest_Event {
	this := &BatchRequest_Event{}
	this.EventType = that.GetEventType()
	this.Event = that.GetEvent()
	return this
}

type DependencyFace interface {
	GetName() string
	GetVersion() string
	GetHomepage() string
	GetSource() *Dependency_Source
}

func NewDependencyFromFace(that DependencyFace) *Dependency {
	this := &Dependency{}
	this.Name = that.GetName()
	this.Version = that.GetVersion()
	this.Homepage = that.GetHomepage()
	this.Source = that.GetSource()
	return this
}

type Dependency_SourceFace interface {
	GetName() string
	GetRemotes() []string
}

func NewDependency_SourceFromFace(that Dependency_SourceFace) *Dependency_Source {
	this := &Dependency_Source{}
	this.Name = that.GetName()
	this.Remotes = that.GetRemotes()
	return this
}

type RequestRecordFace interface {
	GetVersion() string
	GetRulespackId() string
	GetClientIp() string
	GetRequest() RequestRecord_Request
	GetResponse() RequestRecord_Response
	GetObserved() RequestRecord_Observed
}

func NewRequestRecordFromFace(that RequestRecordFace) *RequestRecord {
	this := &RequestRecord{}
	this.Version = that.GetVersion()
	this.RulespackId = that.GetRulespackId()
	this.ClientIp = that.GetClientIp()
	this.Request = that.GetRequest()
	this.Response = that.GetResponse()
	this.Observed = that.GetObserved()
	return this
}

type RequestRecord_RequestFace interface {
	GetRid() string
	GetHeaders() []RequestRecord_Request_Header
	GetVerb() string
	GetPath() string
	GetRawPath() string
	GetHost() string
	GetPort() string
	GetRemoteIp() string
	GetRemotePort() string
	GetScheme() string
	GetUserAgent() string
	GetReferer() string
	GetParameters() RequestRecord_Request_Parameters
}

func NewRequestRecord_RequestFromFace(that RequestRecord_RequestFace) *RequestRecord_Request {
	return &RequestRecord_Request{
		Rid:        that.GetRid(),
		Headers:    that.GetHeaders(),
		Verb:       that.GetVerb(),
		Path:       that.GetPath(),
		RawPath:    that.GetRawPath(),
		Host:       that.GetHost(),
		Port:       that.GetPort(),
		RemoteIp:   that.GetRemoteIp(),
		RemotePort: that.GetRemotePort(),
		Scheme:     that.GetScheme(),
		UserAgent:  that.GetUserAgent(),
		Referer:    that.GetReferer(),
		Parameters: that.GetParameters(),
	}
}

type RequestRecord_Request_HeaderFace interface {
	GetKey() string
	GetValue() string
}

func NewRequestRecord_Request_HeaderFromFace(that RequestRecord_Request_HeaderFace) *RequestRecord_Request_Header {
	this := &RequestRecord_Request_Header{}
	this.Key = that.GetKey()
	this.Value = that.GetValue()
	return this
}

type RequestRecord_ResponseFace interface {
	GetStatus() uint32
	GetContentLength() uint32
	GetContentType() string
}

func NewRequestRecord_ResponseFromFace(that RequestRecord_ResponseFace) *RequestRecord_Response {
	this := &RequestRecord_Response{}
	this.Status = that.GetStatus()
	this.ContentLength = that.GetContentLength()
	this.ContentType = that.GetContentType()
	return this
}

type RequestRecord_ObservedFace interface {
	GetAttacks() []*RequestRecord_Observed_Attack
	GetSdk() []*RequestRecord_Observed_SDKEvent
	GetSqreenExceptions() []*RequestRecord_Observed_Exception
	GetObservations() []*RequestRecord_Observed_Observation
	GetDataPoints() []*RequestRecord_Observed_DataPoint
}

func NewRequestRecord_ObservedFromFace(that RequestRecord_ObservedFace) *RequestRecord_Observed {
	this := &RequestRecord_Observed{}
	this.Attacks = that.GetAttacks()
	this.Sdk = that.GetSdk()
	this.SqreenExceptions = that.GetSqreenExceptions()
	this.Observations = that.GetObservations()
	this.DataPoints = that.GetDataPoints()
	return this
}

type RequestRecord_Observed_AttackFace interface {
	GetRuleName() string
	GetTest() bool
	GetInfo() interface{}
	GetTime() time.Time
	GetBlock() bool
}

func NewRequestRecord_Observed_AttackFromFace(that RequestRecord_Observed_AttackFace) *RequestRecord_Observed_Attack {
	this := &RequestRecord_Observed_Attack{}
	this.RuleName = that.GetRuleName()
	this.Test = that.GetTest()
	this.Time = that.GetTime()
	this.Block = that.GetBlock()
	this.Info = that.GetInfo()
	return this
}

type RequestRecord_Observed_SDKEventFace interface {
	GetTime() time.Time
	GetName() string
	GetArgs() RequestRecord_Observed_SDKEvent_Args
}

func NewRequestRecord_Observed_SDKEventFromFace(that RequestRecord_Observed_SDKEventFace) *RequestRecord_Observed_SDKEvent {
	this := &RequestRecord_Observed_SDKEvent{}
	this.Time = that.GetTime()
	this.Name = that.GetName()
	this.Args = that.GetArgs()
	return this
}

type RequestRecord_Observed_SDKEvent_Args_TrackFace interface {
	GetEvent() string
	GetOptions() *RequestRecord_Observed_SDKEvent_Args_Track_Options
}

func NewRequestRecord_Observed_SDKEvent_Args_TrackFromFace(that RequestRecord_Observed_SDKEvent_Args_TrackFace) *RequestRecord_Observed_SDKEvent_Args_Track {
	this := &RequestRecord_Observed_SDKEvent_Args_Track{}
	this.Event = that.GetEvent()
	this.Options = that.GetOptions()
	return this
}

type RequestRecord_Observed_SDKEvent_Args_Track_OptionsFace interface {
	GetProperties() *Struct
	GetUserIdentifiers() *Struct
}

func NewRequestRecord_Observed_SDKEvent_Args_Track_OptionsFromFace(that RequestRecord_Observed_SDKEvent_Args_Track_OptionsFace) *RequestRecord_Observed_SDKEvent_Args_Track_Options {
	this := &RequestRecord_Observed_SDKEvent_Args_Track_Options{}
	this.Properties = that.GetProperties()
	this.UserIdentifiers = that.GetUserIdentifiers()
	return this
}

type RequestRecord_Observed_SDKEvent_Args_IdentifyFace interface {
	GetUserIdentifiers() *Struct
}

func NewRequestRecord_Observed_SDKEvent_Args_IdentifyFromFace(that RequestRecord_Observed_SDKEvent_Args_IdentifyFace) *RequestRecord_Observed_SDKEvent_Args_Identify {
	this := &RequestRecord_Observed_SDKEvent_Args_Identify{}
	this.UserIdentifiers = that.GetUserIdentifiers()
	return this
}

type RequestRecord_Observed_ExceptionFace interface {
	GetMessage() string
	GetKlass() string
	GetRuleName() string
	GetTest() bool
	GetInfos() string
	GetBacktrace() []string
	GetTime() time.Time
}

func NewRequestRecord_Observed_ExceptionFromFace(that RequestRecord_Observed_ExceptionFace) *RequestRecord_Observed_Exception {
	this := &RequestRecord_Observed_Exception{}
	this.Message = that.GetMessage()
	this.Klass = that.GetKlass()
	this.RuleName = that.GetRuleName()
	this.Test = that.GetTest()
	this.Infos = that.GetInfos()
	this.Backtrace = that.GetBacktrace()
	this.Time = that.GetTime()
	return this
}

type RequestRecord_Observed_ObservationFace interface {
	GetCategory() string
	GetKey() string
	GetValue() string
	GetTime() time.Time
}

func NewRequestRecord_Observed_ObservationFromFace(that RequestRecord_Observed_ObservationFace) *RequestRecord_Observed_Observation {
	this := &RequestRecord_Observed_Observation{}
	this.Category = that.GetCategory()
	this.Key = that.GetKey()
	this.Value = that.GetValue()
	this.Time = that.GetTime()
	return this
}

type RequestRecord_Observed_DataPointFace interface {
	GetRulespackId() string
	GetRuleName() string
	GetTime() time.Time
	GetInfos() string
}

func NewRequestRecord_Observed_DataPointFromFace(that RequestRecord_Observed_DataPointFace) *RequestRecord_Observed_DataPoint {
	this := &RequestRecord_Observed_DataPoint{}
	this.RulespackId = that.GetRulespackId()
	this.RuleName = that.GetRuleName()
	this.Time = that.GetTime()
	this.Infos = that.GetInfos()
	return this
}

type ActionsPackResponse struct {
	Actions []ActionsPackResponse_Action `protobuf:"bytes,1,rep,name=actions,proto3" json:"actions"`
}

type ActionsPackResponse_Action struct {
	ActionId     string                            `protobuf:"bytes,1,opt,name=action_id,json=actionId,proto3" json:"action_id"`
	Action       string                            `protobuf:"bytes,2,opt,name=action,proto3" json:"action"`
	Duration     float64                           `json:"duration"`
	SendResponse bool                              `protobuf:"varint,4,opt,name=send_response,json=sendResponse,proto3" json:"send_response"`
	Parameters   ActionsPackResponse_Action_Params `protobuf:"bytes,5,opt,name=parameters,proto3" json:"parameters"`
}

type ActionsPackResponse_Action_Params struct {
	Url    string              `protobuf:"bytes,1,opt,name=url,proto3" json:"url"`
	Users  []map[string]string `proto3" json:"users"`
	IpCidr []string            `protobuf:"bytes,3,rep,name=ip_cidr,json=ipCidr,proto3" json:"ip_cidr"`
}

type BlockedIPEventProperties struct {
	ActionId string                          `protobuf:"bytes,1,opt,name=action_id,json=actionId,proto3" json:"action_id,omitempty"`
	Output   BlockedIPEventProperties_Output `protobuf:"bytes,2,opt,name=output,proto3" json:"output"`
}

type BlockedIPEventProperties_Output struct {
	IpAddress string `protobuf:"bytes,1,opt,name=ip_address,json=ipAddress,proto3" json:"ip_address,omitempty"`
}

func NewBlockedIPEventPropertiesFromFace(that BlockedIPEventPropertiesFace) *BlockedIPEventProperties {
	this := &BlockedIPEventProperties{}
	this.ActionId = that.GetActionId()
	this.Output = that.GetOutput()
	return this
}

type BlockedIPEventPropertiesFace interface {
	GetActionId() string
	GetOutput() BlockedIPEventProperties_Output
}

type BlockedIPEventProperties_OutputFace interface {
	GetIpAddress() string
}

func NewBlockedIPEventProperties_OutputFromFace(that BlockedIPEventProperties_OutputFace) *BlockedIPEventProperties_Output {
	this := &BlockedIPEventProperties_Output{}
	this.IpAddress = that.GetIpAddress()
	return this
}

type BlockedUserEventProperties struct {
	ActionId string                           `json:"action_id"`
	Output   BlockedUserEventPropertiesOutput `json:"output"`
}

type BlockedUserEventPropertiesOutput struct {
	User map[string]string `json:"user"`
}

func NewBlockedUserEventPropertiesFromFace(that BlockedUserEventPropertiesFace) *BlockedUserEventProperties {
	this := &BlockedUserEventProperties{}
	this.ActionId = that.GetActionId()
	this.Output = that.GetOutput()
	return this
}

type BlockedUserEventPropertiesFace interface {
	GetActionId() string
	GetOutput() BlockedUserEventPropertiesOutput
}

type BlockedUserEventPropertiesOutputFace interface {
	GetUser() map[string]string
}

func NewBlockedUserEventPropertiesOutputFromFace(that BlockedUserEventPropertiesOutputFace) *BlockedUserEventPropertiesOutput {
	this := &BlockedUserEventPropertiesOutput{}
	this.User = that.GetUser()
	return this
}

type RedirectedIPEventProperties struct {
	ActionId string                            `json:"action_id,omitempty"`
	Output   RedirectedIPEventPropertiesOutput `json:"output"`
}

type RedirectedIPEventPropertiesOutput struct {
	IpAddress string `json:"ip_address"`
	URL       string `json:"url"`
}

func NewRedirectedIPEventPropertiesFromFace(that RedirectedIPEventPropertiesFace) *RedirectedIPEventProperties {
	return &RedirectedIPEventProperties{
		ActionId: that.GetActionId(),
		Output:   that.GetOutput(),
	}
}

type RedirectedIPEventPropertiesFace interface {
	GetActionId() string
	GetOutput() RedirectedIPEventPropertiesOutput
}

type RedirectedIPEventPropertiesOutputFace interface {
	GetIpAddress() string
	GetURL() string
}

func NewRedirectedIPEventPropertiesOutputFromFace(that RedirectedIPEventPropertiesOutputFace) *RedirectedIPEventPropertiesOutput {
	return &RedirectedIPEventPropertiesOutput{
		IpAddress: that.GetIpAddress(),
		URL:       that.GetURL(),
	}
}

type RedirectedUserEventProperties struct {
	ActionId string                              `json:"action_id"`
	Output   RedirectedUserEventPropertiesOutput `json:"output"`
}

type RedirectedUserEventPropertiesOutput struct {
	User map[string]string `json:"user"`
}

func NewRedirectedUserEventPropertiesFromFace(that RedirectedUserEventPropertiesFace) *RedirectedUserEventProperties {
	return &RedirectedUserEventProperties{
		ActionId: that.GetActionId(),
		Output:   that.GetOutput(),
	}
}

type RedirectedUserEventPropertiesFace interface {
	GetActionId() string
	GetOutput() RedirectedUserEventPropertiesOutput
}

type RedirectedUserEventPropertiesOutputFace interface {
	GetUser() map[string]string
}

func NewRedirectedUserEventPropertiesOutputFromFace(that RedirectedUserEventPropertiesOutputFace) *RedirectedUserEventPropertiesOutput {
	this := &RedirectedUserEventPropertiesOutput{}
	this.User = that.GetUser()
	return this
}

type RulesPackResponse struct {
	PackID string `json:"pack_id"`
	Rules  []Rule `json:"rules"`
}

type AppBundle struct {
	Signature    string          `json:"bundle_signature"`
	Dependencies []AppDependency `json:"dependencies"`
}

type AppDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

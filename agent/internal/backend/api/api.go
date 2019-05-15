// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api

import (
	"encoding/json"
	"time"
	//	_ "github.com/gogo/protobuf/gogoproto"
	github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
	//	_ "github.com/gogo/protobuf/types"
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
	Time time.Time `protobuf:"bytes,1,opt,name=time,proto3,stdtime" json:"time"`
	Pid  uint32    `protobuf:"varint,3,opt,name=pid,proto3" json:"pid"`
	Ppid uint32    `protobuf:"varint,4,opt,name=ppid,proto3" json:"ppid"`
	Euid uint32    `protobuf:"varint,5,opt,name=euid,proto3" json:"euid"`
	Egid uint32    `protobuf:"varint,6,opt,name=egid,proto3" json:"egid"`
	Uid  uint32    `protobuf:"varint,7,opt,name=uid,proto3" json:"uid"`
	Gid  uint32    `protobuf:"varint,8,opt,name=gid,proto3" json:"gid"`
	Name string    `protobuf:"bytes,9,opt,name=name,proto3" json:"name"`
}

type AppLoginResponse struct {
	Error     string                   `json:"error"`
	SessionId string                   `protobuf:"bytes,1,opt,name=session_id,json=sessionId,proto3" json:"session_id"`
	Status    bool                     `protobuf:"varint,2,opt,name=status,proto3" json:"status"`
	Commands  []CommandRequest         `protobuf:"bytes,3,rep,name=commands,proto3" json:"commands"`
	Features  AppLoginResponse_Feature `protobuf:"bytes,4,opt,name=features,proto3" json:"features"`
	PackId    string                   `protobuf:"bytes,5,opt,name=pack_id,json=packId,proto3" json:"pack_id"`
	Rules     []Rule                   `protobuf:"bytes,6,rep,name=rules,proto3" json:"rules"`
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
	CommandResults map[string]CommandResult `protobuf:"bytes,1,rep,name=command_results,json=commandResults,proto3" json:"command_results,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Metrics        []MetricResponse         `protobuf:"bytes,2,rep,name=metrics,proto3" json:"metrics"`
}

type AppBeatResponse struct {
	Commands []CommandRequest `protobuf:"bytes,1,rep,name=commands,proto3" json:"commands"`
	Status   bool             `protobuf:"varint,2,opt,name=status,proto3" json:"status"`
}

type BatchRequest struct {
	Batch []BatchRequest_Event `json:"batch"`
}

type RequestRecordEvent RequestRecord

func (*RequestRecordEvent) GetEventType() string {
	return "request_record"
}

func (e *RequestRecordEvent) GetEvent() Struct {
	return Struct{e}
}

type ExceptionEvent struct {
	Time        time.Time        `json:"time"`
	Klass       string           `json:"klass"`
	Message     string           `json:"message"`
	RulespackID string           `json:"rulespack_id"`
	Context     ExceptionContext `json:"context"`
}

type ExceptionEventFace interface {
	GetTime() time.Time
	GetKlass() string
	GetMessage() string
	GetRulespackID() string
	GetContext() ExceptionContext
}

func NewExceptionEventFromFace(e ExceptionEventFace) *ExceptionEvent {
	return &ExceptionEvent{
		Time:        e.GetTime(),
		Klass:       e.GetKlass(),
		Message:     e.GetMessage(),
		Context:     e.GetContext(),
		RulespackID: e.GetRulespackID(),
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

type RequestRecord_Request struct {
	Rid        string                         `protobuf:"bytes,1,opt,name=rid,proto3" json:"rid"`
	Headers    []RequestRecord_Request_Header `protobuf:"bytes,2,rep,name=headers,proto3" json:"headers"`
	Verb       string                         `protobuf:"bytes,3,opt,name=verb,proto3" json:"verb"`
	Path       string                         `protobuf:"bytes,4,opt,name=path,proto3" json:"path"`
	RawPath    string                         `protobuf:"bytes,5,opt,name=raw_path,json=rawPath,proto3" json:"raw_path"`
	Host       string                         `protobuf:"bytes,6,opt,name=host,proto3" json:"host"`
	Port       string                         `protobuf:"bytes,7,opt,name=port,proto3" json:"port"`
	RemoteIp   string                         `protobuf:"bytes,8,opt,name=remote_ip,json=remoteIp,proto3" json:"remote_ip"`
	RemotePort string                         `protobuf:"bytes,9,opt,name=remote_port,json=remotePort,proto3" json:"remote_port"`
	Scheme     string                         `protobuf:"bytes,10,opt,name=scheme,proto3" json:"scheme"`
	UserAgent  string                         `protobuf:"bytes,11,opt,name=user_agent,json=userAgent,proto3" json:"user_agent"`
	Referer    string                         `protobuf:"bytes,12,opt,name=referer,proto3" json:"referer"`
	Params     RequestRecord_Request_Params   `protobuf:"bytes,13,opt,name=params,proto3" json:"params"`
}

type RequestRecord_Request_Header struct {
	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value"`
}

type RequestRecord_Request_Params struct {
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
	RuleName  string    `protobuf:"bytes,1,opt,name=rule_name,json=ruleName,proto3" json:"rule_name"`
	Test      bool      `protobuf:"varint,2,opt,name=test,proto3" json:"test"`
	Infos     string    `protobuf:"bytes,3,opt,name=infos,proto3" json:"infos"`
	Backtrace []string  `protobuf:"bytes,4,rep,name=backtrace,proto3" json:"backtrace"`
	Time      time.Time `protobuf:"bytes,5,opt,name=time,proto3,stdtime" json:"time"`
	Block     bool      `protobuf:"varint,6,opt,name=block,proto3" json:"block"`
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

func (m *RequestRecord_Observed_SDKEvent_Args) GetArgs() isRequestRecord_Observed_SDKEvent_Args_Args {
	if m != nil {
		return m.Args
	}
	return nil
}

func (m *RequestRecord_Observed_SDKEvent_Args) GetTrack() *RequestRecord_Observed_SDKEvent_Args_Track {
	if x, ok := m.GetArgs().(*RequestRecord_Observed_SDKEvent_Args_Track_); ok {
		return x.Track
	}
	return nil
}

func (m *RequestRecord_Observed_SDKEvent_Args) GetIdentify() *RequestRecord_Observed_SDKEvent_Args_Identify {
	if x, ok := m.GetArgs().(*RequestRecord_Observed_SDKEvent_Args_Identify_); ok {
		return x.Identify
	}
	return nil
}

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

func (this *AppLoginRequest) GetBundleSignature() string {
	return this.BundleSignature
}

func (this *AppLoginRequest) GetVariousInfos() AppLoginRequest_VariousInfos {
	return this.VariousInfos
}

func (this *AppLoginRequest) GetAgentType() string {
	return this.AgentType
}

func (this *AppLoginRequest) GetAgentVersion() string {
	return this.AgentVersion
}

func (this *AppLoginRequest) GetOsType() string {
	return this.OsType
}

func (this *AppLoginRequest) GetHostname() string {
	return this.Hostname
}

func (this *AppLoginRequest) GetRuntimeType() string {
	return this.RuntimeType
}

func (this *AppLoginRequest) GetRuntimeVersion() string {
	return this.RuntimeVersion
}

func (this *AppLoginRequest) GetFrameworkType() string {
	return this.FrameworkType
}

func (this *AppLoginRequest) GetFrameworkVersion() string {
	return this.FrameworkVersion
}

func (this *AppLoginRequest) GetEnvironment() string {
	return this.Environment
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
}

func (this *AppLoginRequest_VariousInfos) GetTime() time.Time {
	return this.Time
}

func (this *AppLoginRequest_VariousInfos) GetPid() uint32 {
	return this.Pid
}

func (this *AppLoginRequest_VariousInfos) GetPpid() uint32 {
	return this.Ppid
}

func (this *AppLoginRequest_VariousInfos) GetEuid() uint32 {
	return this.Euid
}

func (this *AppLoginRequest_VariousInfos) GetEgid() uint32 {
	return this.Egid
}

func (this *AppLoginRequest_VariousInfos) GetUid() uint32 {
	return this.Uid
}

func (this *AppLoginRequest_VariousInfos) GetGid() uint32 {
	return this.Gid
}

func (this *AppLoginRequest_VariousInfos) GetName() string {
	return this.Name
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
	return this
}

type AppLoginResponseFace interface {
	GetSessionId() string
	GetStatus() bool
	GetCommands() []CommandRequest
	GetFeatures() AppLoginResponse_Feature
	GetPackId() string
	GetRules() []Rule
}

func (this *AppLoginResponse) GetSessionId() string {
	return this.SessionId
}

func (this *AppLoginResponse) GetStatus() bool {
	return this.Status
}

func (this *AppLoginResponse) GetCommands() []CommandRequest {
	return this.Commands
}

func (this *AppLoginResponse) GetFeatures() AppLoginResponse_Feature {
	return this.Features
}

func (this *AppLoginResponse) GetPackId() string {
	return this.PackId
}

func (this *AppLoginResponse) GetRules() []Rule {
	return this.Rules
}

func NewAppLoginResponseFromFace(that AppLoginResponseFace) *AppLoginResponse {
	this := &AppLoginResponse{}
	this.SessionId = that.GetSessionId()
	this.Status = that.GetStatus()
	this.Commands = that.GetCommands()
	this.Features = that.GetFeatures()
	this.PackId = that.GetPackId()
	this.Rules = that.GetRules()
	return this
}

type AppLoginResponse_FeatureFace interface {
	Proto() github_com_gogo_protobuf_proto.Message
	GetBatchSize() uint32
	GetMaxStaleness() uint32
	GetHeartbeatDelay() uint32
}

func (this *AppLoginResponse_Feature) GetBatchSize() uint32 {
	return this.BatchSize
}

func (this *AppLoginResponse_Feature) GetMaxStaleness() uint32 {
	return this.MaxStaleness
}

func (this *AppLoginResponse_Feature) GetHeartbeatDelay() uint32 {
	return this.HeartbeatDelay
}

func NewAppLoginResponse_FeatureFromFace(that AppLoginResponse_FeatureFace) *AppLoginResponse_Feature {
	this := &AppLoginResponse_Feature{}
	this.BatchSize = that.GetBatchSize()
	this.MaxStaleness = that.GetMaxStaleness()
	this.HeartbeatDelay = that.GetHeartbeatDelay()
	return this
}

type CommandResultFace interface {
	Proto() github_com_gogo_protobuf_proto.Message
	GetOutput() string
	GetStatus() bool
}

func (this *CommandResult) GetOutput() string {
	return this.Output
}

func (this *CommandResult) GetStatus() bool {
	return this.Status
}

func NewCommandResultFromFace(that CommandResultFace) *CommandResult {
	this := &CommandResult{}
	this.Output = that.GetOutput()
	this.Status = that.GetStatus()
	return this
}

type MetricResponseFace interface {
	Proto() github_com_gogo_protobuf_proto.Message
	GetName() string
	GetStart() time.Time
	GetFinish() time.Time
	GetObservation() Struct
}

func (this *MetricResponse) GetName() string {
	return this.Name
}

func (this *MetricResponse) GetStart() time.Time {
	return this.Start
}

func (this *MetricResponse) GetFinish() time.Time {
	return this.Finish
}

func (this *MetricResponse) GetObservation() Struct {
	return this.Observation
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

func (this *AppBeatRequest) GetCommandResults() map[string]CommandResult {
	return this.CommandResults
}

func (this *AppBeatRequest) GetMetrics() []MetricResponse {
	return this.Metrics
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

func (this *AppBeatResponse) GetCommands() []CommandRequest {
	return this.Commands
}

func (this *AppBeatResponse) GetStatus() bool {
	return this.Status
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

func (this *BatchRequest) GetBatch() []BatchRequest_Event {
	return this.Batch
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

func (this *BatchRequest_Event) GetEventType() string {
	return this.EventType
}

func (this *BatchRequest_Event) GetEvent() Struct {
	return this.Event
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

func (this *Dependency) GetName() string {
	return this.Name
}

func (this *Dependency) GetVersion() string {
	return this.Version
}

func (this *Dependency) GetHomepage() string {
	return this.Homepage
}

func (this *Dependency) GetSource() *Dependency_Source {
	return this.Source
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

func (this *Dependency_Source) GetName() string {
	return this.Name
}

func (this *Dependency_Source) GetRemotes() []string {
	return this.Remotes
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

func (this *RequestRecord) GetVersion() string {
	return this.Version
}

func (this *RequestRecord) GetRulespackId() string {
	return this.RulespackId
}

func (this *RequestRecord) GetClientIp() string {
	return this.ClientIp
}

func (this *RequestRecord) GetRequest() RequestRecord_Request {
	return this.Request
}

func (this *RequestRecord) GetResponse() RequestRecord_Response {
	return this.Response
}

func (this *RequestRecord) GetObserved() RequestRecord_Observed {
	return this.Observed
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
	GetParams() RequestRecord_Request_Params
}

func (this *RequestRecord_Request) GetRid() string {
	return this.Rid
}

func (this *RequestRecord_Request) GetHeaders() []RequestRecord_Request_Header {
	return this.Headers
}

func (this *RequestRecord_Request) GetVerb() string {
	return this.Verb
}

func (this *RequestRecord_Request) GetPath() string {
	return this.Path
}

func (this *RequestRecord_Request) GetRawPath() string {
	return this.RawPath
}

func (this *RequestRecord_Request) GetHost() string {
	return this.Host
}

func (this *RequestRecord_Request) GetPort() string {
	return this.Port
}

func (this *RequestRecord_Request) GetRemoteIp() string {
	return this.RemoteIp
}

func (this *RequestRecord_Request) GetRemotePort() string {
	return this.RemotePort
}

func (this *RequestRecord_Request) GetScheme() string {
	return this.Scheme
}

func (this *RequestRecord_Request) GetUserAgent() string {
	return this.UserAgent
}

func (this *RequestRecord_Request) GetReferer() string {
	return this.Referer
}

func (this *RequestRecord_Request) GetParams() RequestRecord_Request_Params {
	return this.Params
}

func NewRequestRecord_RequestFromFace(that RequestRecord_RequestFace) *RequestRecord_Request {
	this := &RequestRecord_Request{}
	this.Rid = that.GetRid()
	this.Headers = that.GetHeaders()
	this.Verb = that.GetVerb()
	this.Path = that.GetPath()
	this.RawPath = that.GetRawPath()
	this.Host = that.GetHost()
	this.Port = that.GetPort()
	this.RemoteIp = that.GetRemoteIp()
	this.RemotePort = that.GetRemotePort()
	this.Scheme = that.GetScheme()
	this.UserAgent = that.GetUserAgent()
	this.Referer = that.GetReferer()
	this.Params = that.GetParams()
	return this
}

type RequestRecord_Request_HeaderFace interface {
	GetKey() string
	GetValue() string
}

func (this *RequestRecord_Request_Header) GetKey() string {
	return this.Key
}

func (this *RequestRecord_Request_Header) GetValue() string {
	return this.Value
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

func (this *RequestRecord_Response) GetStatus() uint32 {
	return this.Status
}

func (this *RequestRecord_Response) GetContentLength() uint32 {
	return this.ContentLength
}

func (this *RequestRecord_Response) GetContentType() string {
	return this.ContentType
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

func (this *RequestRecord_Observed) GetAttacks() []*RequestRecord_Observed_Attack {
	return this.Attacks
}

func (this *RequestRecord_Observed) GetSdk() []*RequestRecord_Observed_SDKEvent {
	return this.Sdk
}

func (this *RequestRecord_Observed) GetSqreenExceptions() []*RequestRecord_Observed_Exception {
	return this.SqreenExceptions
}

func (this *RequestRecord_Observed) GetObservations() []*RequestRecord_Observed_Observation {
	return this.Observations
}

func (this *RequestRecord_Observed) GetDataPoints() []*RequestRecord_Observed_DataPoint {
	return this.DataPoints
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
	GetInfos() string
	GetBacktrace() []string
	GetTime() time.Time
	GetBlock() bool
}

func (this *RequestRecord_Observed_Attack) GetRuleName() string {
	return this.RuleName
}

func (this *RequestRecord_Observed_Attack) GetTest() bool {
	return this.Test
}

func (this *RequestRecord_Observed_Attack) GetInfos() string {
	return this.Infos
}

func (this *RequestRecord_Observed_Attack) GetBacktrace() []string {
	return this.Backtrace
}

func (this *RequestRecord_Observed_Attack) GetTime() time.Time {
	return this.Time
}

func (this *RequestRecord_Observed_Attack) GetBlock() bool {
	return this.Block
}

func NewRequestRecord_Observed_AttackFromFace(that RequestRecord_Observed_AttackFace) *RequestRecord_Observed_Attack {
	this := &RequestRecord_Observed_Attack{}
	this.RuleName = that.GetRuleName()
	this.Test = that.GetTest()
	this.Infos = that.GetInfos()
	this.Backtrace = that.GetBacktrace()
	this.Time = that.GetTime()
	this.Block = that.GetBlock()
	return this
}

type RequestRecord_Observed_SDKEventFace interface {
	GetTime() time.Time
	GetName() string
	GetArgs() RequestRecord_Observed_SDKEvent_Args
}

func (this *RequestRecord_Observed_SDKEvent) GetTime() time.Time {
	return this.Time
}

func (this *RequestRecord_Observed_SDKEvent) GetName() string {
	return this.Name
}

func (this *RequestRecord_Observed_SDKEvent) GetArgs() RequestRecord_Observed_SDKEvent_Args {
	return this.Args
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

func (this *RequestRecord_Observed_SDKEvent_Args_Track) GetEvent() string {
	return this.Event
}

func (this *RequestRecord_Observed_SDKEvent_Args_Track) GetOptions() *RequestRecord_Observed_SDKEvent_Args_Track_Options {
	return this.Options
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

func (this *RequestRecord_Observed_SDKEvent_Args_Track_Options) GetProperties() *Struct {
	return this.Properties
}

func (this *RequestRecord_Observed_SDKEvent_Args_Track_Options) GetUserIdentifiers() *Struct {
	return this.UserIdentifiers
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

func (this *RequestRecord_Observed_SDKEvent_Args_Identify) GetUserIdentifiers() *Struct {
	return this.UserIdentifiers
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

func (this *RequestRecord_Observed_Exception) GetMessage() string {
	return this.Message
}

func (this *RequestRecord_Observed_Exception) GetKlass() string {
	return this.Klass
}

func (this *RequestRecord_Observed_Exception) GetRuleName() string {
	return this.RuleName
}

func (this *RequestRecord_Observed_Exception) GetTest() bool {
	return this.Test
}

func (this *RequestRecord_Observed_Exception) GetInfos() string {
	return this.Infos
}

func (this *RequestRecord_Observed_Exception) GetBacktrace() []string {
	return this.Backtrace
}

func (this *RequestRecord_Observed_Exception) GetTime() time.Time {
	return this.Time
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

func (this *RequestRecord_Observed_Observation) GetCategory() string {
	return this.Category
}

func (this *RequestRecord_Observed_Observation) GetKey() string {
	return this.Key
}

func (this *RequestRecord_Observed_Observation) GetValue() string {
	return this.Value
}

func (this *RequestRecord_Observed_Observation) GetTime() time.Time {
	return this.Time
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

func (this *RequestRecord_Observed_DataPoint) GetRulespackId() string {
	return this.RulespackId
}

func (this *RequestRecord_Observed_DataPoint) GetRuleName() string {
	return this.RuleName
}

func (this *RequestRecord_Observed_DataPoint) GetTime() time.Time {
	return this.Time
}

func (this *RequestRecord_Observed_DataPoint) GetInfos() string {
	return this.Infos
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
	ActionId string                            `json:"action_id"`
	Output   BlockedUserEventProperties_Output `json:"output"`
}

type BlockedUserEventProperties_Output struct {
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
	GetOutput() BlockedUserEventProperties_Output
}

type BlockedUserEventProperties_OutputFace interface {
	GetUser() map[string]string
}

func NewBlockedUserEventProperties_OutputFromFace(that BlockedUserEventProperties_OutputFace) *BlockedUserEventProperties_Output {
	this := &BlockedUserEventProperties_Output{}
	this.User = that.GetUser()
	return this
}

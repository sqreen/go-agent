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
	BundleSignature  string                       `json:"bundle_signature"`
	VariousInfos     AppLoginRequest_VariousInfos `json:"various_infos"`
	AgentType        string                       `json:"agent_type"`
	AgentVersion     string                       `json:"agent_version"`
	OsType           string                       `json:"os_type"`
	Hostname         string                       `json:"hostname"`
	RuntimeType      string                       `json:"runtime_type"`
	RuntimeVersion   string                       `json:"runtime_version"`
	FrameworkType    string                       `json:"framework_type"`
	FrameworkVersion string                       `json:"framework_version"`
	Environment      string                       `json:"environment"`
}

type AppLoginRequest_VariousInfos struct {
	Time             time.Time `json:"time"`
	Pid              uint32    `json:"pid"`
	Ppid             uint32    `json:"ppid"`
	Euid             uint32    `json:"euid"`
	Egid             uint32    `json:"egid"`
	Uid              uint32    `json:"uid"`
	Gid              uint32    `json:"gid"`
	Name             string    `json:"name"`
	LibSqreenVersion *string   `json:"libsqreen_version"`
	HasDependencies  bool      `json:"has_dependencies"`
	HasLibsqreen     bool      `json:"has_libsqreen"`
}

type AppLoginResponse struct {
	Error     string                   `json:"error"`
	SessionId string                   `json:"session_id"`
	Status    bool                     `json:"status"`
	Commands  []CommandRequest         `json:"commands"`
	Features  AppLoginResponse_Feature `json:"features"`
	PackId    string                   `json:"pack_id"`
}

type AppLoginResponse_Feature struct {
	BatchSize      uint32 `json:"batch_size"`
	MaxStaleness   uint32 `json:"max_staleness"`
	HeartbeatDelay uint32 `json:"heartbeat_delay"`
	UseSignals     bool   `json:"use_signals"`
}

type CommandRequest struct {
	Name string `json:"name"`
	Uuid string `json:"uuid"`
	// Params: parse and validate the params when used.
	Params []json.RawMessage `json:"params"`
}

type CommandResult struct {
	Output string `json:"output"`
	Status bool   `json:"status"`
}

type MetricResponse struct {
	Name        string    `json:"name"`
	Start       time.Time `json:"start"`
	Finish      time.Time `json:"finish"`
	Observation Struct    `json:"observation"`
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
	LineNumber int    `json:"line_number"`
}

type StackFrameFace interface {
	GetMethod() string
	GetFile() string
	GetLineNumber() int
}

func NewStackFrameFromFace(e StackFrameFace) *StackFrame {
	return &StackFrame{
		Method:     e.GetMethod(),
		File:       e.GetFile(),
		LineNumber: e.GetLineNumber()}
}

type BatchRequest_Event struct {
	EventType string `json:"event_type"`
	Event     Struct `json:"event"`
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
	Beta       bool               `json:"beta"`
}

type RuleConditions struct{}
type RuleCallbacks map[string][]string

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
	Strategy string                   `json:"strategy,omitempty"`
	Klass    string                   `json:"klass"`
	Method   string                   `json:"method"`
	Callback string                   `json:"callback_class"`
	Config   *ReflectedCallbackConfig `json:"arguments_options"`
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

type ReflectedCallbackBindingAccessorConfig struct {
	Capabilities []string `json:"capabilities"`
}

type ReflectedCallbackBlockStrategyReturnFunctionErrorConfig struct {
	RetIndex uint `json:"ret_index"`
}

type ReflectedCallbackBlockStrategyConfig struct {
	Type string `json:"type"`
	ReflectedCallbackBlockStrategyReturnFunctionErrorConfig
}

type ReflectedCallbackHTTPProtectionContextFromFuncArgConfig struct {
	ArgIndex uint `json:"arg_index"`
}

type ReflectedCallbackHTTPProtectionContextConfig struct {
	Type string `json:"type"`
	ReflectedCallbackHTTPProtectionContextFromFuncArgConfig
}

type ReflectedCallbackHTTPProtectionConfig struct {
	BlockStrategy ReflectedCallbackBlockStrategyConfig `json:"block_strategy"`
}

type ReflectedCallbackProtectionConfig struct {
	Type string `json:"type"`
	ReflectedCallbackHTTPProtectionConfig
}

type ReflectedCallbackConfig struct {
	Type            string                                 `json:"type"`
	Protection      *ReflectedCallbackProtectionConfig     `json:"protection"`
	BindingAccessor ReflectedCallbackBindingAccessorConfig `json:"binding_accessor"`
}

type Dependency struct {
	Name     string             `json:"name"`
	Version  string             `json:"version"`
	Homepage string             `json:"homepage"`
	Source   *Dependency_Source `json:"source"`
}

type Dependency_Source struct {
	Name    string   `json:"name"`
	Remotes []string `json:"remotes"`
}

type RequestRecord struct {
	Version     string                 `json:"version"`
	RulespackId string                 `json:"rulespack_id"`
	ClientIp    string                 `json:"client_ip"`
	Request     RequestRecord_Request  `json:"request"`
	Response    RequestRecord_Response `json:"response"`
	Observed    RequestRecord_Observed `json:"observed"`
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
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RequestRecord_Request_Parameters struct {
	// Query parameters
	Query map[string][]string `json:"query,omitempty"`
	// application/x-www-form-urlencoded or multipart/form-data parameters
	Form      map[string][]string `json:"form,omitempty"`
	Framework map[string][]string `json:"framework,omitempty"`
}

type RequestRecord_Response struct {
	Status        int    `json:"status"`
	ContentLength int64  `json:"content_length"`
	ContentType   string `json:"content_type"`
}

type RequestRecord_Observed struct {
	Attacks []*RequestRecord_Observed_Attack   `json:"attacks,omitempty"`
	Sdk     []*RequestRecord_Observed_SDKEvent `json:"sdk,omitempty"`
}

type RequestRecord_Observed_Attack struct {
	RuleName  string       `json:"rule_name"`
	Test      bool         `json:"test"`
	Info      interface{}  `json:"infos"`
	Time      time.Time    `json:"time"`
	Block     bool         `json:"block"`
	Backtrace []StackFrame `json:"backtrace,omitempty"`
	Beta      bool         `json:"beta"`
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
	Time time.Time                            `json:"time"`
	Name string                               `json:"name"`
	Args RequestRecord_Observed_SDKEvent_Args `json:"args"`
}

// Helper message type to disable the face extension only on it and not in
// the entire SDKEvent message type. oneof + face is not supported.
type RequestRecord_Observed_SDKEvent_Args struct {
	// Types that are valid to be assigned to Args:
	//	*RequestRecord_Observed_SDKEvent_Args_Track_
	//	*RequestRecord_Observed_SDKEvent_Args_Identify_
	Args isRequestRecord_Observed_SDKEvent_Args_Args
}

type isRequestRecord_Observed_SDKEvent_Args_Args interface {
	isRequestRecord_Observed_SDKEvent_Args_Args()
}

type RequestRecord_Observed_SDKEvent_Args_Track_ struct {
	Track *RequestRecord_Observed_SDKEvent_Args_Track
}
type RequestRecord_Observed_SDKEvent_Args_Identify_ struct {
	Identify *RequestRecord_Observed_SDKEvent_Args_Identify
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
	Event   string                                              `json:"event"`
	Options *RequestRecord_Observed_SDKEvent_Args_Track_Options `json:"options"`
}

type RequestRecord_Observed_SDKEvent_Args_Track_Options struct {
	Properties      *Struct           `json:"properties,omitempty"`
	UserIdentifiers map[string]string `json:"user_identifiers,omitempty"`
}

// Serialized into:
// [ <user_identifiers> ]
type RequestRecord_Observed_SDKEvent_Args_Identify struct {
	UserIdentifiers map[string]string `json:"user_identifiers"`
}

type RequestRecord_Observed_Exception struct {
	Message   string    `json:"message"`
	Klass     string    `json:"klass"`
	RuleName  string    `json:"rule_name"`
	Test      bool      `json:"test"`
	Infos     string    `json:"infos"`
	Backtrace []string  `json:"backtrace"`
	Time      time.Time `json:"time"`
}

type RequestRecord_Observed_Observation struct {
	Category string    `json:"category"`
	Key      string    `json:"key"`
	Value    string    `json:"value"`
	Time     time.Time `json:"time"`
}

type RequestRecord_Observed_DataPoint struct {
	RulespackId string    `json:"rulespack_id"`
	RuleName    string    `json:"rule_name"`
	Time        time.Time `json:"time"`
	Infos       string    `json:"infos"`
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
	GetHasDependencies() bool
	GetHasLibsqreen() bool
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
	this.HasDependencies = that.GetHasDependencies()
	this.HasLibsqreen = that.GetHasLibsqreen()
	return this
}

type AppLoginResponseFace interface {
	GetSessionId() string
	GetStatus() bool
	GetCommands() []CommandRequest
	GetFeatures() AppLoginResponse_Feature
	GetPackId() string
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

type AppBeatRequestFace interface {
	GetCommandResults() map[string]CommandResult
	GetMetrics() []MetricResponse
}

type AppBeatResponseFace interface {
	GetCommands() []CommandRequest
	GetStatus() bool
}

type BatchRequestFace interface {
	GetBatch() []BatchRequest_Event
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

type Dependency_SourceFace interface {
	GetName() string
	GetRemotes() []string
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

type RequestRecord_ResponseFace interface {
	GetStatus() int
	GetContentLength() int64
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
	return this
}

type RequestRecord_Observed_AttackFace interface {
	GetRuleName() string
	GetTest() bool
	GetInfo() interface{}
	GetTime() time.Time
	GetBlock() bool
	GetBacktrace() []StackFrame
}

func NewRequestRecord_Observed_AttackFromFace(that RequestRecord_Observed_AttackFace) *RequestRecord_Observed_Attack {
	this := &RequestRecord_Observed_Attack{}
	this.RuleName = that.GetRuleName()
	this.Test = that.GetTest()
	this.Time = that.GetTime()
	this.Block = that.GetBlock()
	this.Info = that.GetInfo()
	this.Backtrace = that.GetBacktrace()
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
	GetUserIdentifiers() map[string]string
}

func NewRequestRecord_Observed_SDKEvent_Args_Track_OptionsFromFace(that RequestRecord_Observed_SDKEvent_Args_Track_OptionsFace) *RequestRecord_Observed_SDKEvent_Args_Track_Options {
	this := &RequestRecord_Observed_SDKEvent_Args_Track_Options{}
	this.Properties = that.GetProperties()
	this.UserIdentifiers = that.GetUserIdentifiers()
	return this
}

type RequestRecord_Observed_SDKEvent_Args_IdentifyFace interface {
	GetUserIdentifiers() map[string]string
}

func NewRequestRecord_Observed_SDKEvent_Args_IdentifyFromFace(that RequestRecord_Observed_SDKEvent_Args_IdentifyFace) *RequestRecord_Observed_SDKEvent_Args_Identify {
	this := &RequestRecord_Observed_SDKEvent_Args_Identify{}
	this.UserIdentifiers = that.GetUserIdentifiers()
	return this
}

type ActionsPackResponse struct {
	Actions []ActionsPackResponse_Action `json:"actions"`
}

type ActionsPackResponse_Action struct {
	ActionId     string                            `json:"action_id"`
	Action       string                            `json:"action"`
	Duration     float64                           `json:"duration"`
	SendResponse bool                              `json:"send_response"`
	Parameters   ActionsPackResponse_Action_Params `json:"parameters"`
}

type ActionsPackResponse_Action_Params struct {
	Url    string              `json:"url"`
	Users  []map[string]string `json:"users"`
	IpCidr []string            `json:"ip_cidr"`
}

type BlockedIPEventProperties struct {
	ActionId string                          `json:"action_id,omitempty"`
	Output   BlockedIPEventProperties_Output `json:"output"`
}

type BlockedIPEventProperties_Output struct {
	IpAddress string `json:"ip_address,omitempty"`
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

type AgentMessage struct {
	Id      string `json:"id"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	fuzz := fuzz.New().NilChance(0).Funcs(FuzzStruct)

	for _, tc := range []struct {
		Object     interface{}
		NewMessage interface{}
	}{
		{
			Object:     &ExceptionContext{},
			NewMessage: api.NewExceptionContextFromFace,
		},
		// TODO: Write fuzzer
		//{
		//	Object:     &ExceptionEvent{},
		//	NewMessage: api.NewExceptionEventFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &AppLoginRequest{},
		//	NewMessage: api.NewAppLoginRequestFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &AppLoginRequest_VariousInfos{},
		//	NewMessage: api.NewAppLoginRequest_VariousInfosFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &AppLoginResponse{},
		//	NewMessage: api.NewAppLoginResponseFromFace,
		//},
		{
			Object:     &AppLoginResponse_Feature{},
			NewMessage: api.NewAppLoginResponse_FeatureFromFace,
		},
		{
			Object:     &CommandResult{},
			NewMessage: api.NewCommandResultFromFace,
		},
		// TODO: Write fuzzer
		//{
		//	Object:     &MetricResponse{},
		//	NewMessage: api.NewMetricResponseFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &AppBeatRequest{},
		//	NewMessage: api.NewAppBeatRequestFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &AppBeatResponse{},
		//	NewMessage: api.NewAppBeatResponseFromFace,
		//},
		// TODO: Write fuzzer
		//{
		//	Object:     &BatchRequest{},
		//	NewMessage: api.NewBatchRequestFromFace,
		//},
	} {
		tc := tc // new scope
		t.Run(fmt.Sprintf("%T", tc.Object), func(t *testing.T) {
			// Random object value
			fuzz.Fuzz(tc.Object)
			// Call the api message constructor from the interface
			msg := reflect.ValueOf(tc.NewMessage).Call([]reflect.Value{reflect.ValueOf(tc.Object)})[0].Interface()
			// It should be equal to the original object value
			// The following line assume the message type is wrapped in an adapter
			// type that implements the message interface and therefore converts it
			// to the same type as the API message type to be able to compare them.
			obj := reflect.ValueOf(tc.Object).Convert(reflect.TypeOf(msg)).Interface()
			require.Equal(t, obj, msg)
			// Marshal it to JSON
			buf, err := json.Marshal(msg)
			require.NoError(t, err)
			// Create another pointer to the same type
			msg2 := reflect.New(reflect.TypeOf(msg).Elem()).Interface()
			// Unmarshal the JSON buffer
			err = json.Unmarshal(buf, msg2)
			require.NoError(t, err)
			// Messages should be equal
			require.Equal(t, msg, msg2)
		})
	}
}

type ExceptionContext api.ExceptionContext

func (o ExceptionContext) GetBacktrace() []api.StackFrame {
	return o.Backtrace
}

type ExceptionEvent api.ExceptionEvent

func (o *ExceptionEvent) GetTime() time.Time {
	return o.Time
}

func (o *ExceptionEvent) GetKlass() string {
	return o.Klass
}

func (o *ExceptionEvent) GetMessage() string {
	return o.Message
}

func (o *ExceptionEvent) GetRulespackID() string {
	return o.RulespackID
}

func (o *ExceptionEvent) GetContext() api.ExceptionContext {
	return o.Context
}

type AppLoginRequest api.AppLoginRequest

func (this *AppLoginRequest) GetBundleSignature() string {
	return this.BundleSignature
}

func (this *AppLoginRequest) GetVariousInfos() api.AppLoginRequest_VariousInfos {
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

type AppLoginRequest_VariousInfos api.AppLoginRequest_VariousInfos

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

type AppLoginResponse api.AppLoginResponse

func (this *AppLoginResponse) GetSessionId() string {
	return this.SessionId
}

func (this *AppLoginResponse) GetStatus() bool {
	return this.Status
}

func (this *AppLoginResponse) GetCommands() []api.CommandRequest {
	return this.Commands
}

func (this *AppLoginResponse) GetFeatures() api.AppLoginResponse_Feature {
	return this.Features
}

func (this *AppLoginResponse) GetPackId() string {
	return this.PackId
}

type AppLoginResponse_Feature api.AppLoginResponse_Feature

func (this *AppLoginResponse_Feature) GetBatchSize() uint32 {
	return this.BatchSize
}

func (this *AppLoginResponse_Feature) GetMaxStaleness() uint32 {
	return this.MaxStaleness
}

func (this *AppLoginResponse_Feature) GetHeartbeatDelay() uint32 {
	return this.HeartbeatDelay
}

type CommandResult api.CommandResult

func (this *CommandResult) GetOutput() string {
	return this.Output
}

func (this *CommandResult) GetStatus() bool {
	return this.Status
}

type MetricResponse api.MetricResponse

func (this *MetricResponse) GetName() string {
	return this.Name
}

func (this *MetricResponse) GetStart() time.Time {
	return this.Start
}

func (this *MetricResponse) GetFinish() time.Time {
	return this.Finish
}

func (this *MetricResponse) GetObservation() api.Struct {
	return this.Observation
}

type AppBeatRequest api.AppBeatRequest

func (this *AppBeatRequest) GetCommandResults() map[string]api.CommandResult {
	return this.CommandResults
}

func (this *AppBeatRequest) GetMetrics() []api.MetricResponse {
	return this.Metrics
}

type AppBeatResponse api.AppBeatResponse

func (this *AppBeatResponse) GetCommands() []api.CommandRequest {
	return this.Commands
}

func (this *AppBeatResponse) GetStatus() bool {
	return this.Status
}

type BatchRequest api.BatchRequest

func (this *BatchRequest) GetBatch() []api.BatchRequest_Event {
	return this.Batch
}

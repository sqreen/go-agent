// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// This package defines agent-specific signals.
package signal

import (
	"strconv"
	"strings"
	"time"

	legacy_api "github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-sdk/signal/client/api"
	http_trace "github.com/sqreen/go-sdk/signal/http"
)

func FormatAgentSource(agentVersion string) string {
	var source strings.Builder
	source.WriteString("sqreen:agent:golang:")
	source.WriteString(agentVersion)
	return source.String()
}

type AgentMessagePayload struct {
	Hash    string                 `json:"hash"`
	Message string                 `json:"message"`
	Infos   map[string]interface{} `json:"infos,omitempty"`
}

func newAgentMessagePayload(hash string, message string, infos map[string]interface{}) *api.SignalPayload {
	return api.NewPayload("agent_message/2020-01-01T00:00:00.000Z", AgentMessagePayload{
		Hash:    hash,
		Message: message,
		Infos:   infos,
	})
}

type AgentInfra struct {
	AgentType      string `json:"agent_type"`
	AgentVersion   string `json:"agent_version"`
	OSType         string `json:"os_type"`
	Hostname       string `json:"hostname"`
	RuntimeVersion string `json:"runtime_version"`
}

func NewAgentInfra(agentVersion, osType, hostname, runtimeVersion string) *AgentInfra {
	return &AgentInfra{
		AgentType:      "golang",
		AgentVersion:   agentVersion,
		OSType:         osType,
		Hostname:       hostname,
		RuntimeVersion: runtimeVersion,
	}
}

func fromLegacyRequestRecord(record *legacy_api.RequestRecord, infra *AgentInfra) (*http_trace.Trace, error) {
	// Parse the port numbers by ignoring parsing errors and keeping the default
	// zero value otherwise anyway to avoid dropping the the trace for that error.
	// For example, the port number can be possibly empty.
	port, _ := strconv.ParseUint(record.Request.Port, 10, 64)
	remotePort, _ := strconv.ParseUint(record.Request.RemotePort, 10, 64)

	headers := make([][]string, len(record.Request.Headers))
	for i, e := range record.Request.Headers {
		headers[i] = []string{e.Key, e.Value}
	}

	// TODO: get the request timestamp?
	t := time.Now()
	req := http_trace.NewRequestContext(record.Request.Rid, headers, record.Request.UserAgent, record.Request.Scheme, record.Request.Verb, record.Request.Host, record.Request.RemoteIp, record.Request.Path, record.Request.Referer, port, remotePort, record.Request.Parameters)
	resp := http_trace.NewResponseContext(record.Response.Status, record.Response.ContentType, record.Response.ContentLength)
	traceCtx := http_trace.NewContext(req, resp)

	var (
		// The global user id can be set with the Identify SDK method. It needs to be
		// globally set in the actor field of the HTTP trace if any.
		globalUserID map[string]string

		// The set of signals to add to the HTTP trace
		signals []*api.Signal
	)

	// Convert SDK events
	for _, e := range record.Observed.Sdk {
		switch e.Name {
		case event.SDKMethodIdentify:
			if actual, ok := e.Args.Args.(*legacy_api.RequestRecord_Observed_SDKEvent_Args_Identify_); ok {
				// Globally identified user with the identify sdk method (request-global).
				// As a signal, it now goes into the HTTP trace actor struct field and is
				// no longer in the list of events.
				globalUserID = actual.Identify.UserIdentifiers
			}

		case event.SDKMethodTrack:
			if actual, ok := e.Args.Args.(*legacy_api.RequestRecord_Observed_SDKEvent_Args_Track_); ok {
				signal := fromLegacyTrackEvent(actual.Track, e.Time)
				signals = append(signals, (*api.Signal)(signal))
			}
		}
	}

	// Convert attacks
	for _, a := range record.Observed.Attacks {
		attack := fromLegacyAttack(a, record.RulespackId)
		signals = append(signals, (*api.Signal)(attack))
	}

	actor := http_trace.NewActor([]string{record.ClientIp}, record.Request.UserAgent, globalUserID)

	// The trace can be now created. Note that the source is not set so that it
	// doesn't overwrite
	trace := http_trace.NewTrace("", t, actor, infra, traceCtx, signals)

	return trace, nil
}

type Attack api.Point

func fromLegacyAttack(a *legacy_api.RequestRecord_Observed_Attack, rulePackID string) *Attack {
	var name, source strings.Builder

	name.WriteString("sq.agent.attack.")
	name.WriteString(a.AttackType)

	source.WriteString("sqreen:rule:")
	source.WriteString(rulePackID)
	source.WriteString(":")
	source.WriteString(a.RuleName)

	payload := newAttackPayload(a.Test, a.Block, a.Info)
	return (*Attack)(api.NewPoint(name.String(), source.String(), a.Time, nil, nil, nil, fromLegacyBacktrace(a.Backtrace), nil, payload))
}

type AttackPayload struct {
	Test  bool        `json:"test"`
	Block bool        `json:"block"`
	Infos interface{} `json:"infos"`
}

func newAttackPayload(test, block bool, infos interface{}) *api.SignalPayload {
	return api.NewPayload("attack/2020-01-01T00:00:00.000Z", AttackPayload{
		Test:  test,
		Block: block,
		Infos: infos,
	})
}

func fromLegacyTrackEvent(track *legacy_api.RequestRecord_Observed_SDKEvent_Args_Track, t time.Time) *api.Point {
	var name strings.Builder
	name.WriteString("sq.sdk.")
	name.WriteString(track.Event)
	return api.NewPoint(name.String(), "sqreen:sdk:track", t, nil, nil, nil, nil, nil, newTrackEventPayload(track.Options.Properties, track.Options.UserIdentifiers))
}

type TrackEventPayload struct {
	Properties      *legacy_api.Struct `json:"properties,omitempty"`
	UserIdentifiers map[string]string  `json:"user_identifiers,omitempty"`
}

func newTrackEventPayload(properties *legacy_api.Struct, userIdentifiers map[string]string) *api.SignalPayload {
	// TODO:  change the Struct to a proper type once we remove the legacy
	//  objects
	return api.NewPayload("track_event/2020-01-01T00:00:00.000Z", TrackEventPayload{
		Properties:      properties,
		UserIdentifiers: userIdentifiers,
	})
}

func FromLegacyBatch(b []legacy_api.BatchRequest_Event, infra *AgentInfra, logger plog.ErrorLogger) api.Batch {
	batch := make(api.Batch, 0, len(b))
	for i := range b {
		var signal api.SignalFace
		switch evt := b[i].Event.Value.(type) {
		case legacy_api.RequestRecordEvent:
			trace, err := fromLegacyRequestRecord(evt.RequestRecord, infra)
			if err != nil {
				logger.Error(sqerrors.WithInfo(sqerrors.Wrap(err, "could not create the HTTP trace"), evt))
				continue
			}
			signal = trace

		case *legacy_api.ExceptionEvent:
			exception := fromLegacyAgentException(evt, infra)
			signal = (*api.Point)(exception)

		default:
			logger.Error(sqerrors.Errorf("unexpected batch event type `%T`", b))
			continue
		}
		batch = append(batch, signal)
	}

	return batch
}

type AgentException api.Point

func fromLegacyAgentException(e *legacy_api.ExceptionEvent, infra *AgentInfra) *AgentException {
	source := FormatAgentSource(infra.AgentVersion)

	payload := newAgentExceptionPayload(e.Klass, e.Message, e.Infos)
	location := fromLegacyBacktrace(e.Context.Backtrace)
	return (*AgentException)(api.NewPoint("sq.agent.exception", source, e.Time, nil, nil, infra, location, nil, payload))
}

type (
	StackTraceLocation struct {
		StackTrace []StackFrame `json:"stack_trace,omitempty"`
	}

	StackFrame struct {
		InApp    bool   `json:"in_app"`
		Function string `json:"function"`
		AbsPath  string `json:"abs_path"`
		LineNo   int    `json:"lineno"`
	}
)

func fromLegacyBacktrace(bt []legacy_api.StackFrame) StackTraceLocation {
	frames := make([]StackFrame, len(bt))
	for i, f := range bt {
		frames[i] = StackFrame{
			InApp:    true,
			Function: f.Method,
			AbsPath:  f.File,
			LineNo:   f.LineNumber,
		}
	}

	return StackTraceLocation{
		StackTrace: frames,
	}
}

type AgentExceptionPayload struct {
	Klass   string      `json:"klass"`
	Message string      `json:"message"`
	Infos   interface{} `json:"infos,omitempty"`
}

func newAgentExceptionPayload(klass, message string, infos interface{}) *api.SignalPayload {
	return api.NewPayload("exception/2020-01-01T00:00:00.000Z", AgentExceptionPayload{
		Klass:   klass,
		Message: message,
		Infos:   infos,
	})
}

func FromLegacyMetrics(metrics []legacy_api.MetricResponse, agentVersion string, logger plog.ErrorLogger) api.Batch {
	batch := make(api.Batch, len(metrics))
	for i, metric := range metrics {
		metric, err := convertLegacyMetrics(&metric, agentVersion)
		if err != nil {
			logger.Error(err)
		} else {
			batch[i] = metric
		}
	}
	return batch
}

func convertLegacyMetrics(metric *legacy_api.MetricResponse, agentVersion string) (*api.Metric, error) {
	var name strings.Builder
	name.WriteString("sq.agent.metric.")
	name.WriteString(metric.Name)

	source := FormatAgentSource(agentVersion)

	values, ok := metric.Observation.Value.(map[string]int64)
	if !ok {
		return nil, sqerrors.Errorf("unexpected type of metric values `%T` instead of `%T`", metric.Observation.Value, values)
	}

	return api.NewSumMetric(name.String(), source, metric.Start, metric.Finish, metric.Finish.Sub(metric.Start), values), nil
}

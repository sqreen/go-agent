// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package internal

import (
	"encoding/json"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/app"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-libsqreen/waf"
)

type closedHTTPRequestContextEventAPIAdapter struct {
	adaptee            *closedHTTPRequestContextEvent
	stripHTTPReferer   bool
	httpClientIPHeader string
}

func newProtectedHTTPRequestEventAPIAdapter(event *closedHTTPRequestContextEvent, stripHTTPReferer bool, httpClientIPHeader string) *closedHTTPRequestContextEventAPIAdapter {
	return &closedHTTPRequestContextEventAPIAdapter{
		adaptee:            event,
		stripHTTPReferer:   stripHTTPReferer,
		httpClientIPHeader: httpClientIPHeader,
	}
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetVersion() string {
	return api.RequestRecordVersion
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetRulespackId() string {
	return a.adaptee.rulepackID
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetClientIp() string {
	return a.adaptee.request.ClientIP().String()
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetStart() time.Time {
	return a.adaptee.start
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetEnd() time.Time {
	return a.adaptee.finish
}

type httpRequestAPIAdapter struct {
	adaptee            types.RequestReader
	stripHTTPReferer   bool
	httpClientIPHeader string
}

func (a *httpRequestAPIAdapter) GetRid() string {
	if rid := a.adaptee.Header("X-Request-Id"); rid != nil {
		return *rid
	}
	return ""
}

func (a *httpRequestAPIAdapter) GetHeaders() []api.RequestRecord_Request_Header {
	trackedHeaders := config.TrackedHTTPHeaders
	if extraHeader := a.httpClientIPHeader; extraHeader != "" {
		trackedHeaders = append(trackedHeaders, extraHeader)
	}
	var headers []api.RequestRecord_Request_Header
	for _, header := range trackedHeaders {
		if value := a.adaptee.Header(header); value != nil {
			headers = append(headers, api.RequestRecord_Request_Header{
				Key:   header,
				Value: *value,
			})
		}
	}
	return headers
}

func (a *httpRequestAPIAdapter) GetVerb() string {
	return a.adaptee.Method()
}

func (a *httpRequestAPIAdapter) GetPath() string {
	return a.adaptee.URL().Path
}

func (a *httpRequestAPIAdapter) GetHost() string {
	return a.adaptee.Host()
}

func (a *httpRequestAPIAdapter) GetPort() string {
	_, port, _ := net.SplitHostPort(a.adaptee.Host())
	return port
}

func (a *httpRequestAPIAdapter) GetRemoteIp() string {
	ip, _, _ := net.SplitHostPort(a.adaptee.RemoteAddr())
	return ip
}

func (a *httpRequestAPIAdapter) GetRemotePort() string {
	_, port, _ := net.SplitHostPort(a.adaptee.RemoteAddr())
	return port
}

func (a *httpRequestAPIAdapter) GetScheme() string {
	if a.adaptee.IsTLS() {
		return "https"
	} else {
		return "http"
	}
}

func (a *httpRequestAPIAdapter) GetUserAgent() string {
	return a.adaptee.UserAgent()
}

func (a *httpRequestAPIAdapter) GetReferer() string {
	if a.stripHTTPReferer {
		return ""
	}
	return a.adaptee.Referer()
}

func (a *httpRequestAPIAdapter) GetParameters() api.RequestRecord_Request_Parameters {
	req := a.adaptee
	// .Form and .PostForm are taken as is, without calling `ParseForm()` so
	// that we take what has been done during the request handling.
	// So they can be nil even if there were form parameters in the
	// body.
	var rawBody string
	if len(req.Body()) > 0 {
		rawBody = "<Redacted By Sqreen>"
	}
	return api.RequestRecord_Request_Parameters{
		Query:   req.QueryForm(),
		Form:    req.PostForm(),
		Params:  req.Params(),
		RawBody: rawBody,
	}
}

func (a closedHTTPRequestContextEventAPIAdapter) GetRequest() api.RequestRecord_Request {
	return *api.NewRequestRecord_RequestFromFace(&httpRequestAPIAdapter{
		adaptee:            a.adaptee.request,
		stripHTTPReferer:   a.stripHTTPReferer,
		httpClientIPHeader: a.httpClientIPHeader,
	})
}

type httpResponseAPIAdapter struct {
	adaptee types.ResponseFace
}

func (a httpResponseAPIAdapter) GetStatus() int {
	return a.adaptee.Status()
}

func (a httpResponseAPIAdapter) GetContentLength() int64 {
	return a.adaptee.ContentLength()
}

func (a httpResponseAPIAdapter) GetContentType() string {
	return a.adaptee.ContentType()
}

func (a closedHTTPRequestContextEventAPIAdapter) GetResponse() api.RequestRecord_Response {
	return *api.NewRequestRecord_ResponseFromFace(httpResponseAPIAdapter{adaptee: a.adaptee.response})
}

type attackEventAPIAdapter event.AttackEvent

func (a *attackEventAPIAdapter) unwrap() *event.AttackEvent {
	return (*event.AttackEvent)(a)
}

func (a *attackEventAPIAdapter) GetAttackType() string {
	return a.unwrap().AttackType
}

func (a *attackEventAPIAdapter) GetRuleName() string {
	return a.unwrap().Rule
}

func (a *attackEventAPIAdapter) GetTest() bool {
	return a.unwrap().Test
}

func (a *attackEventAPIAdapter) GetInfo() interface{} {
	return a.unwrap().Info
}

func (a *attackEventAPIAdapter) GetTime() time.Time {
	return a.unwrap().Timestamp
}

func (a *attackEventAPIAdapter) GetBlock() bool {
	return a.unwrap().Blocked
}

func (a *attackEventAPIAdapter) GetBacktrace() []api.StackFrame {
	return stackTraceAPIAdapter(a.StackTrace).GetBacktrace()
}

type stackTraceAPIAdapter []uintptr

type errorStackTraceAPIAdapter errors.StackTrace

func (a errorStackTraceAPIAdapter) GetBacktrace() []api.StackFrame {
	if len(a) == 0 {
		return nil
	}
	bt := make([]api.StackFrame, len(a))
	for i, f := range a {
		bt[i] = *api.NewStackFrameFromFace(apiStackFrame(f))
	}
	return bt
}

func (a stackTraceAPIAdapter) GetBacktrace() []api.StackFrame {
	if len(a) == 0 {
		return nil
	}
	bt := make([]api.StackFrame, len(a))
	for i, f := range a {
		bt[i] = *api.NewStackFrameFromFace(apiStackFrame(f))
	}
	return bt
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetAttacks() []*api.RequestRecord_Observed_Attack {
	attacks := a.adaptee.events.AttackEvents
	observed := make([]*api.RequestRecord_Observed_Attack, len(attacks))
	for i, attack := range attacks {
		observed[i] = api.NewRequestRecord_Observed_AttackFromFace((*attackEventAPIAdapter)(attack))
	}
	return observed
}

type customEventAPIAdapter event.CustomEvent

func (a *customEventAPIAdapter) GetProperties() *api.Struct {
	if properties := a.unwrap().Properties; properties != nil {
		return &api.Struct{properties}
	}
	return nil
}

func (a *customEventAPIAdapter) GetUserIdentifiers() map[string]string {
	if userID := a.unwrap().UserID; len(userID) != 0 {
		return userID
	}
	return nil
}

func (a *customEventAPIAdapter) GetEvent() string {
	return a.unwrap().Event
}

func (a *customEventAPIAdapter) GetOptions() *api.RequestRecord_Observed_SDKEvent_Args_Track_Options {
	return api.NewRequestRecord_Observed_SDKEvent_Args_Track_OptionsFromFace(a)
}

func (a *customEventAPIAdapter) GetTime() time.Time {
	return a.unwrap().Timestamp
}

func (a *customEventAPIAdapter) GetName() string {
	return a.unwrap().Method
}

func (c *customEventAPIAdapter) GetArgs() (args api.RequestRecord_Observed_SDKEvent_Args) {
	// TODO: change to a type switch
	switch c.unwrap().Method {
	case event.SDKMethodTrack:
		args.Args = &api.RequestRecord_Observed_SDKEvent_Args_Track_{api.NewRequestRecord_Observed_SDKEvent_Args_TrackFromFace(c)}
	case event.SDKMethodIdentify:
		args.Args = &api.RequestRecord_Observed_SDKEvent_Args_Identify_{api.NewRequestRecord_Observed_SDKEvent_Args_IdentifyFromFace(c)}
	}
	return
}

func (a *customEventAPIAdapter) unwrap() *event.CustomEvent {
	return (*event.CustomEvent)(a)
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetObserved() api.RequestRecord_Observed {
	return *api.NewRequestRecord_ObservedFromFace(a)
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetSdk() []*api.RequestRecord_Observed_SDKEvent {
	events := a.adaptee.events.CustomEvents
	observed := make([]*api.RequestRecord_Observed_SDKEvent, len(events))
	for i, e := range events {
		observed[i] = api.NewRequestRecord_Observed_SDKEventFromFace((*customEventAPIAdapter)(e))
	}
	return observed
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetSqreenExceptions() []*api.RequestRecord_Observed_Exception {
	return nil
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetObservations() []*api.RequestRecord_Observed_Observation {
	// TODO with metrics
	return nil
}

func (a *closedHTTPRequestContextEventAPIAdapter) GetDataPoints() []*api.RequestRecord_Observed_DataPoint {
	return nil
}

func newMetricsAPIAdapter(logger plog.ErrorLogger, readyMetrics map[string]metrics.ReadyStore) []api.MetricsTimeBucket {
	if len(readyMetrics) == 0 {
		return nil
	}

	metricsArray := make([]api.MetricsTimeBucket, 0, len(readyMetrics))
	for id, store := range readyMetrics {
		var observation api.MetricsData

		switch store := store.(type) {
		default:
			logger.Error(sqerrors.Errorf("unexpected metrics store type `%T`", store))

		case *metrics.ReadyPerfHistogram:
			values := store.Metrics()
			obs := make(map[string]interface{}, len(values)+1)
			for k, v := range values {
				if v > math.MaxInt64 {
					logger.Error(sqerrors.Errorf("could not marshal to json the uint64 value `%v`: signed int64 representation overflow", k))
					continue
				}
				v := int64(v)

				bucket, ok := k.(metrics.TimeHistogramBucketType)
				if !ok {
					logger.Error(sqerrors.Errorf("unexpected performance bucket value's type `%T`", bucket))
					continue
				}
				key := strconv.FormatUint(uint64(bucket), 10)
				obs[key] = v
			}
			obs["max"] = store.Max()

			observation = api.PerfMetricsData{
				Unit:   store.Unit(),
				Base:   store.Base(),
				Values: obs,
			}

		case *metrics.ReadyTimeHistogram:
			metrics := store.Metrics()
			obs := make(api.SumMetricsData, len(metrics))
			for k, v := range metrics {
				if v > math.MaxInt64 {
					logger.Error(sqerrors.Errorf("could not marshal to json the uint64 value `%v`: signed int64 representation overflow", k))
					continue
				}
				v := int64(v)

				// String keys are directly added
				if s, ok := k.(string); ok {
					obs[s] = v
					continue
				}

				// Non-string keys are serialized into json
				jsonKey, err := json.Marshal(k)
				if err != nil {
					logger.Error(sqerrors.Wrapf(err, "could not marshal to json the key the value `%[1]v` of type `%[1]T`", k))
					continue
				}

				obs[string(jsonKey)] = v
			}

			if len(obs) == 0 {
				continue
			}
			observation = obs
		}

		if observation == nil {
			continue
		}

		metricsArray = append(metricsArray, api.MetricsTimeBucket{
			Name:        id,
			Start:       store.Start(),
			Finish:      store.Finish(),
			Observation: observation,
		})
	}

	return metricsArray
}

type variousInfoAPIAdapter struct {
	*appInfoAPIAdapter
	sqreenDomains api.SqreenDomainStatusMap
}

func (v variousInfoAPIAdapter) GetSqreenDomains() api.SqreenDomainStatusMap {
	return v.sqreenDomains
}

type appInfoAPIAdapter app.Info

func (a *appInfoAPIAdapter) unwrap() *app.Info { return (*app.Info)(a) }

func (a *appInfoAPIAdapter) GetTime() time.Time {
	return a.unwrap().GetProcessInfo().StartTime()
}

func (a *appInfoAPIAdapter) GetPid() uint32 {
	return a.unwrap().GetProcessInfo().Pid()
}

func (a *appInfoAPIAdapter) GetPpid() uint32 {
	return a.unwrap().GetProcessInfo().Ppid()
}

func (a *appInfoAPIAdapter) GetEuid() uint32 {
	return a.unwrap().GetProcessInfo().Euid()
}

func (a *appInfoAPIAdapter) GetEgid() uint32 {
	return a.unwrap().GetProcessInfo().Egid()
}

func (a *appInfoAPIAdapter) GetUid() uint32 {
	return a.unwrap().GetProcessInfo().Uid()
}

func (a *appInfoAPIAdapter) GetGid() uint32 {
	return a.unwrap().GetProcessInfo().Gid()
}

func (a *appInfoAPIAdapter) GetName() string {
	return a.unwrap().GetProcessInfo().Name()
}

func (a *appInfoAPIAdapter) GetLibSqreenVersion() *string {
	return waf.Version()
}

func (a *appInfoAPIAdapter) GetHasDependencies() bool {
	deps, _, _ := a.unwrap().Dependencies()
	return deps != nil
}

func (a *appInfoAPIAdapter) GetHasLibsqreen() bool {
	return a.GetLibSqreenVersion() != nil
}

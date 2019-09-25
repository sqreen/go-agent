// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// CallbackConstructorFunc is a function returning a callback function or a
// CallbackObject configured with the given data.
type CallbacksConstructorFunc func(rule callback.Context, nextProlog sqhook.PrologCallback) (callbackCtor interface{}, err error)

// CallbackObject can be used by callbacks needing to return an object instead
// of a function that will be closed when removed from its hookpoint.
// For example, it allows to release memory out of the garbage collector' scope.
type CallbackObject interface {
	Prolog() sqhook.PrologCallback
	io.Closer
}

// NewCallback returns the callback object or function for the given callback
// name. An error is returned if the callback name is unknown or an error
// occurred during the constructor call.
func NewCallback(name string, rule *CallbackContext, nextProlog sqhook.PrologCallback) (cb interface{}, err error) {
	var callbackCtor CallbacksConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined callback name `%s`", name)
	case "WriteCustomErrorPage":
		callbackCtor = callback.NewWriteCustomErrorPageCallbacks
	case "WriteHTTPRedirection":
		callbackCtor = callback.NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbackCtor = callback.NewAddSecurityHeadersCallbacks
	case "MonitorHTTPStatusCode":
		callbackCtor = callback.NewMonitorHTTPStatusCodeCallbacks
	case "WAF":
		callbackCtor = callback.NewWAFCallback
	}
	return callbackCtor(rule, nextProlog)
}

type CallbackContext struct {
	config              interface{}
	metricsStores       map[string]*metrics.Store
	defaultMetricsStore *metrics.Store
	errorMetricsStore   *metrics.Store
	name                string
	test                bool
	logger              Logger
	plog.ErrorLogger
	blockingMode bool
}

func NewCallbackContext(r *api.Rule, logger Logger, metricsEngine *metrics.Engine, errorMetricsStore *metrics.Store) *CallbackContext {
	config := newCallbackConfig(&r.Data)

	var (
		metricsStores       map[string]*metrics.Store
		defaultMetricsStore *metrics.Store
	)
	if len(r.Metrics) > 0 {
		metricsStores = make(map[string]*metrics.Store)
		for _, m := range r.Metrics {
			metricsStores[m.Name] = metricsEngine.NewStore(m.Name, time.Second*time.Duration(m.Period))
		}
		defaultMetricsStore = metricsStores[r.Metrics[0].Name]
	}

	return &CallbackContext{
		config:              config,
		metricsStores:       metricsStores,
		defaultMetricsStore: defaultMetricsStore,
		errorMetricsStore:   errorMetricsStore,
		name:                r.Name,
		test:                r.Test,
		blockingMode:        r.Block,
		logger:              logger,
		ErrorLogger:         logger,
	}
}

func newCallbackConfig(data *api.RuleData) (config interface{}) {
	if nbData := len(data.Values); nbData == 1 && reflect.TypeOf(data.Values[0].Value).Kind() != reflect.Slice {
		config = data.Values[0].Value
	} else {
		configArray := make([]interface{}, 0, nbData)
		for _, e := range data.Values {
			configArray = append(configArray, e.Value)
		}
		config = configArray
	}
	return config
}

func (d *CallbackContext) Config() interface{} {
	return d.config
}

func (d *CallbackContext) PushMetricsValue(key interface{}, value uint64) {
	err := d.defaultMetricsStore.Add(key, value)
	if err != nil {
		sqErr := sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not add a value to the default metrics store", d.name))
		switch actualErr := err.(type) {
		case metrics.MaxMetricsStoreLengthError:
			d.logger.Debug(sqErr)
			if err := d.errorMetricsStore.Add(actualErr, 1); err != nil {
				d.logger.Debugf("could not update the error metrics store: %v", err)
			}
		default:
			d.logger.Error(sqErr)
		}
	}
}

func (c *CallbackContext) BlockingMode() bool {
	return c.blockingMode
}

// NewAttack creates a new attack based on the rule context and the given
// argument.
func (ctx *CallbackContext) NewAttack(blocked bool, info interface{}) *record.AttackEvent {
	return &record.AttackEvent{
		Rule:      ctx.name,
		Test:      ctx.test,
		Blocked:   blocked,
		Timestamp: time.Now(),
		Info:      info,
	}
}

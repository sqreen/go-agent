// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"fmt"
	"reflect"
	"time"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// CallbackConstructorFunc is a function returning a callback function
// configured with the given data. The data types are known by the constructor
// that can type-assert them.
type CallbacksConstructorFunc func(rule callback.Context, nextProlog sqhook.PrologCallback) (prolog sqhook.PrologCallback, err error)

// NewCallbacks returns the prolog and epilog callbacks of the given callback
// name. And error is returned if the callback name is unknown.
func NewCallbacks(name string, rule *CallbackContext, nextProlog sqhook.PrologCallback) (prolog sqhook.PrologCallback, err error) {
	var callbacksCtor CallbacksConstructorFunc
	switch name {
	default:
		return nil, sqerrors.Errorf("undefined callback name `%s`", name)
	case "WriteCustomErrorPage":
		callbacksCtor = callback.NewWriteCustomErrorPageCallbacks
	case "WriteHTTPRedirection":
		callbacksCtor = callback.NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbacksCtor = callback.NewAddSecurityHeadersCallbacks
	case "MonitorHTTPStatusCode":
		callbacksCtor = callback.NewMonitorHTTPStatusCodeCallbacks
	}
	return callbacksCtor(rule, nextProlog)
}

type CallbackContext struct {
	config              interface{}
	metricsStores       map[string]*metrics.Store
	defaultMetricsStore *metrics.Store
	errorMetricsStore   *metrics.Store
	logger              Logger
	name                string
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
		logger:              logger,
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

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import (
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

// CallbackObject can be used by callbacks needing to return an object instead
// of a function that will be closed when removed from its hookpoint.
// For example, it allows to release memory out of the garbage collector' scope.
//type CallbackObject interface {
//	Prolog() sqhook.PrologCallback
//	io.Closer
//}

type CallbackContext struct {
	metricsStores       map[string]*metrics.Store
	defaultMetricsStore *metrics.Store
	errorMetricsStore   *metrics.Store
	name                string
	testMode            bool
	config              callback.Config
	beta                bool
}

func NewCallbackContext(r *api.Rule, metricsEngine *metrics.Engine, errorMetricsStore *metrics.Store) (*CallbackContext, error) {
	var (
		metricsStores       map[string]*metrics.Store
		defaultMetricsStore *metrics.Store
	)
	if len(r.Metrics) > 0 {
		metricsStores = make(map[string]*metrics.Store)
		for _, m := range r.Metrics {
			metricsStores[m.Name] = metricsEngine.GetStore(m.Name, time.Second*time.Duration(m.Period))
		}
		defaultMetricsStore = metricsStores[r.Metrics[0].Name]
	}

	config, err := newCallbackConfig(r)
	if err != nil {
		return nil, sqerrors.Wrap(err, "callback configuration")
	}

	return &CallbackContext{
		metricsStores:       metricsStores,
		defaultMetricsStore: defaultMetricsStore,
		errorMetricsStore:   errorMetricsStore,
		name:                r.Name,
		testMode:            r.Test,
		beta:                r.Beta,
		config:              config,
	}, nil
}

func (d *CallbackContext) PushMetricsValue(key interface{}, value int64) error {
	err := d.defaultMetricsStore.Add(key, value)
	if err != nil {
		sqErr := sqerrors.Wrapf(err, "rule `%s`: could not add a value to the default metrics store", d.name)
		switch actualErr := err.(type) {
		case metrics.MaxMetricsStoreLengthError:
			if err := d.errorMetricsStore.Add(actualErr, 1); err != nil {
				return sqerrors.Wrap(err, "could not update the error metrics store")
			}
		default:
			return sqErr
		}
	}
	return nil
}

func (d *CallbackContext) Config() callback.Config {
	return d.config
}

func (d *CallbackContext) NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent {
	return &event.AttackEvent{
		Rule:       d.name,
		Test:       d.testMode,
		Beta:       d.beta,
		Blocked:    blocked,
		Timestamp:  time.Now(),
		Info:       info,
		StackTrace: st,
	}
}

type genericCallbackConfig struct {
	config           interface{}
	bindingAccessors []string
	jsFuncs          map[string]string
	callback.Config
}

// Static assert that the callback.GenericCallbackConfig interface is
// implemented.
var _ callback.GenericCallbackConfig = &genericCallbackConfig{}

func (g *genericCallbackConfig) BindingAccessors() []string {
	return g.bindingAccessors
}

func (g *genericCallbackConfig) JSCallbacks(fname string) string {
	return g.jsFuncs[fname]
}

type config struct {
	blockingMode bool
	data         interface{}
	strategy     *api.ReflectedCallbackConfig
}

func (c *config) Strategy() *api.ReflectedCallbackConfig {
	return c.strategy
}

func (c *config) BlockingMode() bool {
	return c.blockingMode
}

func (c *config) Data() interface{} {
	return c.data
}

func newCallbackConfig(r *api.Rule) (callback.Config, error) {
	cfg := &config{
		blockingMode: r.Block,
	}

	var data interface{}
	if nbData := len(r.Data.Values); nbData == 1 && reflect.TypeOf(r.Data.Values[0].Value).Kind() != reflect.Slice {
		data = r.Data.Values[0].Value
	} else {
		dataArray := make([]interface{}, nbData)
		for e := range r.Data.Values {
			dataArray[e] = r.Data.Values[e].Value
		}
		data = dataArray
	}
	cfg.data = data

	if r.Hookpoint.Strategy == "" || r.Hookpoint.Strategy == "native" {
		return cfg, nil
	}

	var (
		bindingAccessors []string
		jsFuncs          map[string]string
	)
	callbacks := r.Callbacks
	jsFuncs = make(map[string]string, len(callbacks))
	for funcName, values := range callbacks {
		nbEntries := len(values)
		if nbEntries == 0 {
			return nil, sqerrors.New("unexpected empty callback configuration entry")
		}

		// the js is stored in the last json array entry
		js := values[nbEntries-1]
		jsFuncs[funcName] = js

		cfg.strategy = r.Hookpoint.Config

		// the rest of it are the binding accessors to compute the js arguments
		bindingAccessors = values[:nbEntries-1]
		// TODO: compile here bindingaccessor.Compile()
	}
	return &genericCallbackConfig{
		Config:           cfg,
		bindingAccessors: bindingAccessors,
		jsFuncs:          jsFuncs,
	}, nil
}

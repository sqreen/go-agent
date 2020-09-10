// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import (
	"fmt"
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/binding-accessor"
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
	metricsStores          map[string]*metrics.Store
	defaultMetricsStore    *metrics.Store
	errorMetricsStore      *metrics.Store
	callCountsMetricsStore *metrics.Store
	preCallCounter         string
	name                   string
	testMode               bool
	config                 callback.NativeCallbackConfig
	attackType             string
}

func NewCallbackContext(r *api.Rule, rulepackID string, metricsEngine *metrics.Engine, errorMetricsStore *metrics.Store) (*CallbackContext, error) {
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

	var (
		callCountsMetricsStore *metrics.Store
		preCallCounter         string
	)
	if r.CallCountInterval != 0 {
		callCountsMetricsStore = metricsEngine.GetStore("sqreen_call_counts", 60*time.Second)
		preCallCounter = fmt.Sprintf("%s/%s/pre", rulepackID, r.Name)
	}

	return &CallbackContext{
		metricsStores:          metricsStores,
		defaultMetricsStore:    defaultMetricsStore,
		errorMetricsStore:      errorMetricsStore,
		name:                   r.Name,
		testMode:               r.Test,
		attackType:             r.AttackType,
		preCallCounter:         preCallCounter,
		callCountsMetricsStore: callCountsMetricsStore,
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

func (d *CallbackContext) NewAttackEvent(blocked bool, info interface{}, st errors.StackTrace) *event.AttackEvent {
	return &event.AttackEvent{
		Rule:       d.name,
		Test:       d.testMode,
		AttackType: d.attackType,
		Blocked:    blocked,
		Timestamp:  time.Now(),
		Info:       info,
		StackTrace: st,
	}
}

func (d *CallbackContext) MonitorPre() {
	// TODO: execution time monitoring and cap
	if d.callCountsMetricsStore != nil {
		if err := d.callCountsMetricsStore.Add(d.preCallCounter, 1); err != nil {
			// TODO: log the error
		}
	}
}

type (
	reflectedCallbackConfig struct {
		callback.NativeCallbackConfig
		strategy *api.ReflectedCallbackConfig
	}

	jsReflectedCallbackConfig struct {
		callback.ReflectedCallbackConfig
		pre  *jsCallbackFuncConfig
		post *jsCallbackFuncConfig
	}

	jsCallbackFuncConfig struct {
		FuncCallParams []bindingaccessor.BindingAccessorFunc
		FuncDecl       *goja.Program
	}
)

// Static assert that interface callback.ReflectedCallbackConfig is implemented.
var _ callback.ReflectedCallbackConfig = &jsReflectedCallbackConfig{}

// Static assert that interface callback.ReflectedCallbackConfig is implemented.
var _ callback.ReflectedCallbackConfig = &reflectedCallbackConfig{}

func (c *reflectedCallbackConfig) Strategy() *api.ReflectedCallbackConfig {
	return c.strategy
}

func (c *jsReflectedCallbackConfig) Pre() (*goja.Program, []bindingaccessor.BindingAccessorFunc) {
	if c.pre == nil {
		return nil, nil
	}
	return c.pre.FuncDecl, c.pre.FuncCallParams
}

func (c *jsReflectedCallbackConfig) Post() (*goja.Program, []bindingaccessor.BindingAccessorFunc) {
	if c.post == nil {
		return nil, nil
	}
	return c.post.FuncDecl, c.post.FuncCallParams
}

type nativeCallbackConfig struct {
	blockingMode bool
	data         interface{}
	strategy     *api.ReflectedCallbackConfig
}

func (c *nativeCallbackConfig) BlockingMode() bool {
	return c.blockingMode
}

func (c *nativeCallbackConfig) Data() interface{} {
	return c.data
}

func newNativeCallbackConfig(r *api.Rule) (callback.NativeCallbackConfig, error) {
	cfg := &nativeCallbackConfig{
		data:         newCallbackConfigData(r.Data.Values),
		blockingMode: r.Block,
	}
	return cfg, nil
}

func newReflectedCallbackConfig(r *api.Rule) (callback.ReflectedCallbackConfig, error) {
	if s := r.Hookpoint.Strategy; s != "reflected" {
		return nil, sqerrors.Errorf("callback config: unexpected hookpoint strategy `%s`", s)
	}
	nativeCfg, err := newNativeCallbackConfig(r)
	if err != nil {
		return nil, err
	}

	return &reflectedCallbackConfig{
		NativeCallbackConfig: nativeCfg,
		strategy:             r.Hookpoint.Config,
	}, nil
}

func newJSReflectedCallbackConfig(r *api.Rule) (callback.JSReflectedCallbackConfig, error) {
	reflectedCfg, err := newReflectedCallbackConfig(r)
	if err != nil {
		return nil, err
	}

	callbacks, ok := r.Callbacks.RuleCallbacksNode.(*api.RuleJSCallbacks)
	if !ok {
		return nil, sqerrors.Errorf("unexpected callbacks type `%T` instead of `%T`", r.Callbacks.RuleCallbacksNode, callbacks)
	}
	pre, err := newJSCallbackFuncConfig("pre", callbacks.Pre)
	if err != nil {
		return nil, err
	}
	post, err := newJSCallbackFuncConfig("post", callbacks.Post)
	if err != nil {
		return nil, err
	}

	if pre == nil && post == nil {
		return nil, sqerrors.New("undefined javascript callbacks `pre` or `post`")
	}

	return &jsReflectedCallbackConfig{
		ReflectedCallbackConfig: reflectedCfg,
		pre:                     pre,
		post:                    post,
	}, nil
}

func newCallbackConfigData(ruleData []api.RuleDataEntry) interface{} {
	l := len(ruleData)

	if l == 0 {
		return nil
	}

	// When the rule data has one entry and is not a slice, return the first
	// entry
	if l == 1 && reflect.TypeOf(ruleData[0].Value).Kind() != reflect.Slice {
		return ruleData[0].Value
	}

	// Otherwise return the unwrapped entries
	dataArray := make([]interface{}, l)
	for i := range ruleData {
		dataArray[i] = ruleData[i].Value
	}
	return dataArray
}

func newJSCallbackFuncConfig(name string, rule []string) (*jsCallbackFuncConfig, error) {
	if len(rule) == 0 {
		return nil, nil
	}

	// The js function definition is stored in the last json array entry, and
	// the rest of it are the binding accessors to evaluate and pass as js func
	// call parameters.
	last := len(rule) - 1
	jsSrc := rule[last]

	// Compile the JS source code
	program, err := goja.Compile(name, jsSrc, true)
	if err != nil {
		return nil, sqerrors.Wrapf(err, "could not compile the js function declaration `%s`", name)
	}

	// Validate it is a non-nil function
	vm := goja.New()
	if _, err := vm.RunProgram(program); err != nil {
		return nil, sqerrors.Wrapf(err, "could not run the js function declaration `%s`", name)
	}
	if f := vm.Get(name); f == nil {
		return nil, sqerrors.Errorf("could not get function definition `%s` from the js vm", name)
	} else if kind := f.ExportType().Kind(); kind != reflect.Func {
		return nil, sqerrors.Errorf("value of `%s` is a `%s` instead of a function", name, kind)
	}

	// Compile the binding accessors to use to get its call parameters
	bindingAccessorSources := rule[:last]
	bindingAccessors, err := compileBindingAccessorExpressions(bindingAccessorSources)
	if err != nil {
		return nil, sqerrors.Wrapf(err, "could not compile the binding accessors of the js function call to `%s`", name)
	}

	return &jsCallbackFuncConfig{
		FuncCallParams: bindingAccessors,
		FuncDecl:       program,
	}, nil
}

func compileBindingAccessorExpressions(bindingAccessors []string) ([]bindingaccessor.BindingAccessorFunc, error) {
	args := make([]bindingaccessor.BindingAccessorFunc, len(bindingAccessors))
	for i, expr := range bindingAccessors {
		ba, err := bindingaccessor.Compile(expr)
		if err != nil {
			return nil, sqerrors.Wrapf(err, "could not compile the binding accessor of argument %d", i)
		}
		args[i] = ba
	}

	return args, nil
}

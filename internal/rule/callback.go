// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import (
	"reflect"
	"time"

	"github.com/dop251/goja"
	"github.com/pkg/errors"
	"github.com/sqreen/go-agent/internal/backend/api"
	bindingaccessor "github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
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
	config              callback.NativeCallbackConfig
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

	return &CallbackContext{
		metricsStores:       metricsStores,
		defaultMetricsStore: defaultMetricsStore,
		errorMetricsStore:   errorMetricsStore,
		name:                r.Name,
		testMode:            r.Test,
		beta:                r.Beta,
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
		Beta:       d.beta,
		Blocked:    blocked,
		Timestamp:  time.Now(),
		Info:       info,
		StackTrace: st,
	}
}

type (
	reflectedCallbackConfig struct {
		pre  *JSCallbackFuncConfig
		post *JSCallbackFuncConfig
		callback.NativeCallbackConfig
		strategy *api.ReflectedCallbackConfig
	}

	JSCallbackFuncConfig struct {
		FuncCallParams []bindingaccessor.BindingAccessorFunc
		FuncDecl       *goja.Program
	}
)

// Static assert that the callback.ReflectedCallbackConfig interface is
// implemented.
var _ callback.ReflectedCallbackConfig = &reflectedCallbackConfig{}

func (c *reflectedCallbackConfig) Pre() (*goja.Program, []bindingaccessor.BindingAccessorFunc) {
	if c.pre == nil {
		return nil, nil
	}
	return c.pre.FuncDecl, c.pre.FuncCallParams
}

func (c *reflectedCallbackConfig) Post() (*goja.Program, []bindingaccessor.BindingAccessorFunc) {
	if c.post == nil {
		return nil, nil
	}
	return c.post.FuncDecl, c.post.FuncCallParams
}

func (c *reflectedCallbackConfig) Strategy() *api.ReflectedCallbackConfig {
	return c.strategy
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
	sqassert.True(r.Hookpoint.Strategy == "reflected")
	nativeCfg, err := newNativeCallbackConfig(r)
	if err != nil {
		return nil, err
	}

	callbacks := r.Callbacks
	pre, err := newJSCallbackFuncConfig("pre", callbacks["pre"])
	if err != nil {
		return nil, err
	}
	post, err := newJSCallbackFuncConfig("post", callbacks["post"])
	if err != nil {
		return nil, err
	}

	if pre == nil && post == nil {
		return nil, sqerrors.New("undefined javascript callbacks `pre` or `post`")
	}

	return &reflectedCallbackConfig{
		NativeCallbackConfig: nativeCfg,
		pre:                  pre,
		post:                 post,
		strategy:             r.Hookpoint.Config,
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

func newJSCallbackFuncConfig(name string, rule []string) (*JSCallbackFuncConfig, error) {
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

	return &JSCallbackFuncConfig{
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

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package rule

import (
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/dop251/goja"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/internal/event"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/rule/callback/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
)

type nativeRuleContext struct {
	name         string
	config       callback.NativeCallbackConfig
	testMode     bool
	blockingMode bool
	critical     bool
	attackType   string
	rulepackID   string
	logger       Logger

	pre  []NativeCallbackMiddlewareFunc
	post []NativeCallbackMiddlewareFunc

	metricsEngine       *metrics.Engine
	metricsStores       map[string]*metrics.TimeHistogram
	defaultMetricsStore *metrics.TimeHistogram

	perfHistogramUnit   float64
	perfHistogramBase   float64
	perfHistogramPeriod time.Duration
}

var _ callback.RuleContext = &nativeRuleContext{}

type (
	NativeCallbackFunc           = func(c callback.CallbackContext)
	NativeCallbackMiddlewareFunc = func(cb NativeCallbackFunc) NativeCallbackFunc
)

func newNativeRuleContext(rule *api.Rule, rulepackID string, metricsEngine *metrics.Engine, logger Logger, perfHistogramUnit, perfHistogramBase float64, perfHistogramPeriod time.Duration) (*nativeRuleContext, error) {
	var (
		metricsStores       map[string]*metrics.TimeHistogram
		defaultMetricsStore *metrics.TimeHistogram
	)
	if len(rule.Metrics) > 0 {
		metricsStores = make(map[string]*metrics.TimeHistogram)
		for _, m := range rule.Metrics {
			metricsStores[m.Name] = metricsEngine.TimeHistogram(m.Name, time.Second*time.Duration(m.Period))
		}
		defaultMetricsStore = metricsStores[rule.Metrics[0].Name]
	}

	r := &nativeRuleContext{
		name:                rule.Name,
		testMode:            rule.Test,
		blockingMode:        rule.Block,
		attackType:          rule.AttackType,
		rulepackID:          rulepackID,
		logger:              logger,
		metricsEngine:       metricsEngine,
		metricsStores:       metricsStores,
		defaultMetricsStore: defaultMetricsStore,
		perfHistogramPeriod: perfHistogramPeriod,
		perfHistogramUnit:   perfHistogramUnit,
		perfHistogramBase:   perfHistogramBase,
	}

	r.buildMiddlewares()

	return r, nil
}

func withPerformanceCap(rule string, overBudgetHistogram *metrics.TimeHistogram) NativeCallbackMiddlewareFunc {
	return func(cb NativeCallbackFunc) NativeCallbackFunc {
		return func(c callback.CallbackContext) {
			if c.ProtectionContext().DeadlineExceeded(0) {
				// TODO: log error once
				_ = overBudgetHistogram.Add(rule, 1)
				return
			}

			cb(c)
		}
	}
}

func withPerformanceMonitoring(perfHistogram *metrics.PerfHistogram) NativeCallbackMiddlewareFunc {
	return func(cb NativeCallbackFunc) NativeCallbackFunc {
		return func(c callback.CallbackContext) {
			sq := c.ProtectionContext().SqreenTime()
			sw := sq.Start()
			defer func() {
				duration := sw.Stop()
				// Compute the milliseconds floating point value out of the nanoseconds
				ms := float64(duration.Nanoseconds()) / float64(time.Millisecond)
				if err := perfHistogram.Add(ms); err != nil {
					// TODO: log once
				}
			}()

			cb(c)
		}
	}
}

func withSafeCall(l plog.ErrorLogger) NativeCallbackMiddlewareFunc {
	return func(cb NativeCallbackFunc) NativeCallbackFunc {
		return func(c callback.CallbackContext) {
			if err := sqsafe.Call(func() error {
				cb(c)
				return nil
			}); err != nil {
				l.Error(err)
			}
		}
	}
}

func withCallCount(store *metrics.TimeHistogram, rule, rulepackID string) NativeCallbackMiddlewareFunc {
	callCounterID := fmt.Sprintf("%s/%s/pre", rulepackID, rule)
	return func(cb NativeCallbackFunc) NativeCallbackFunc {
		return func(c callback.CallbackContext) {
			if err := store.Add(callCounterID, 1); err != nil {
				c.Logger().Error(err)
			}
			cb(c)
		}
	}
}

func (r *nativeRuleContext) Pre(pre NativeCallbackFunc) {
	r.call(pre, r.pre)
}

func (r *nativeRuleContext) Post(post func(c callback.CallbackContext)) {
	r.call(post, r.post)
}

func (r *nativeRuleContext) call(cb NativeCallbackFunc, m []NativeCallbackMiddlewareFunc) {
	c, ok := makeCallbackContext(r)
	if !ok {
		return
	}
	cb = wrapCallback(cb, m)
	cb(c)
}

func (r *nativeRuleContext) SetCritical(critical bool) {
	r.critical = critical
	r.buildMiddlewares()
}

func (r *nativeRuleContext) buildMiddlewares() {
	r.buildPreMiddlewares()
	r.buildPostMiddlewares()
}

func (r *nativeRuleContext) buildPreMiddlewares() {
	perfHist, err := r.metricsEngine.PerfHistogram("sq."+r.name+".pre", r.perfHistogramUnit, r.perfHistogramBase, r.perfHistogramPeriod)
	if err != nil {
		r.logger.Error(sqerrors.Wrap(err, "could not create the performance metrics for the pre callback"))
	}

	var overBudgetHist *metrics.TimeHistogram
	if !r.critical {
		overBudgetHist = r.metricsEngine.TimeHistogram("request_overbudget_cb", r.perfHistogramPeriod)
	}

	callCountHist := r.metricsEngine.TimeHistogram("sqreen_call_counts", r.perfHistogramPeriod)

	r.pre = buildMiddlewares(r, overBudgetHist, perfHist, callCountHist)
}

func (r *nativeRuleContext) buildPostMiddlewares() {
	perfHist, err := r.metricsEngine.PerfHistogram("sq."+r.name+".post", r.perfHistogramUnit, r.perfHistogramBase, r.perfHistogramPeriod)
	if err != nil {
		r.logger.Error(sqerrors.Wrap(err, "could not create the performance metrics for the pre callback"))
	}

	var overBudgetHist *metrics.TimeHistogram
	if r.critical {
		overBudgetHist = r.metricsEngine.TimeHistogram("request_overbudget_cb", r.perfHistogramPeriod)
	}

	callCountHist := r.metricsEngine.TimeHistogram("sqreen_call_counts", r.perfHistogramPeriod)

	r.post = buildMiddlewares(r, overBudgetHist, perfHist, callCountHist)
}

func buildMiddlewares(r *nativeRuleContext, overBudgetHist *metrics.TimeHistogram, perfHist *metrics.PerfHistogram, callCountHist *metrics.TimeHistogram) (m []NativeCallbackMiddlewareFunc) {
	m = append(m, withSafeCall(r.logger))

	if overBudgetHist != nil {
		m = append(m, withPerformanceCap(r.name, overBudgetHist))
	}

	if perfHist != nil {
		m = append(m, withPerformanceMonitoring(perfHist))
	}

	if callCountHist != nil {
		m = append(m, withCallCount(callCountHist, r.name, r.rulepackID))
	}

	return m
}

func wrapCallback(cb NativeCallbackFunc, middlewares []NativeCallbackMiddlewareFunc) NativeCallbackFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		cb = middlewares[i](cb)
	}
	return cb
}

type nativeCallbackContext struct {
	r *nativeRuleContext
	p types.ProtectionContext
}

func makeCallbackContext(r *nativeRuleContext) (c nativeCallbackContext, ok bool) {
	p := types.FromGLS()
	if p == nil {
		ok = false
		return
	}

	return nativeCallbackContext{
		r: r,
		p: p,
	}, true
}

func (c nativeCallbackContext) ProtectionContext() types.ProtectionContext {
	return c.p
}

func (c nativeCallbackContext) Logger() callback.Logger {
	return c.r.logger
}

func (c nativeCallbackContext) AddMetricsValue(key interface{}, value int64) error {
	if err := c.r.defaultMetricsStore.Add(key, value); err != nil {
		sqErr := sqerrors.Wrapf(err, "rule `%s`: could not add a value to the default metrics store", c.r.name)
		switch err.(type) {
		case metrics.MaxMetricsStoreLengthError:
			// TODO: log once
			return nil
		default:
			return sqErr
		}
	}
	return nil
}

func (c nativeCallbackContext) HandleAttack(shouldBlock bool, info interface{}) (blocked bool) {
	block := !c.r.testMode && c.r.blockingMode && shouldBlock
	var attack *event.AttackEvent
	if info != nil {
		attack = &event.AttackEvent{
			Rule:       c.r.name,
			Test:       c.r.testMode,
			Blocked:    block,
			Timestamp:  time.Now(),
			Info:       info,
			StackTrace: callers(),
			AttackType: c.r.attackType,
		}
	}
	return c.p.HandleAttack(block, attack)
}

func callers() []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	return pcs[0:n]
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

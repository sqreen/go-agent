// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

// Package rule implements the engine to manage rules.
//
// Main requirements:
// - Rules can be globally enabled or disabled, independently from setting
//   the list of rules.
// - Rule hookpoints can be undefined, ie. the backend sent more rules than
//   actually required.
// - Errors regarding hookpoint or callbacks should be handled.
// - Setting new rules when already enabled and having active rules should be
//   atomic at the hook level. For example, having a new SQLi rule should not
//   introduce a time when it is disabled, but should instead be replaced with
//   the new one atomically.
package rule

import (
	"crypto/ecdsa"
	"encoding/json"
	"io"
	"io/ioutil"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/config"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/span"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type Engine struct {
	config                               *config.Config
	logger                               plog.DebugLevelLogger
	enabled                              bool
	rulepack                             *rulepack
	metricsEngine                        *metrics.Engine
	publicKey                            *ecdsa.PublicKey
	instrumentationEngine                InstrumentationFace
	perfHistogramUnit, perfHistogramBase float64
	perfHistogramPeriod                  time.Duration
}

type rulepack struct {
	id                string
	hooks             hookDescriptorMap
	reactiveCallbacks []span.EventListener
}

// NewEngine returns a new rule engine.
func NewEngine(cfg *config.Config, logger plog.DebugLevelLogger, instrumentationEngine InstrumentationFace, metricsEngine *metrics.Engine, publicKey *ecdsa.PublicKey, perfHistogramUnit, perfHistogramBase float64, perfHistogramPeriod time.Duration) *Engine {
	if instrumentationEngine == nil {
		instrumentationEngine = defaultInstrumentationEngine
	}

	return &Engine{
		config:                cfg,
		logger:                logger,
		metricsEngine:         metricsEngine,
		publicKey:             publicKey,
		instrumentationEngine: instrumentationEngine,
		perfHistogramBase:     perfHistogramBase,
		perfHistogramUnit:     perfHistogramUnit,
		perfHistogramPeriod:   perfHistogramPeriod,
	}
}

// Health returns a detailed error when the
func (e *Engine) Health(expectedVersion string) error {
	return e.instrumentationEngine.Health(expectedVersion)
}

func (e *Engine) setRulepack(p *rulepack) {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&e.rulepack))
	atomic.StorePointer(addr, unsafe.Pointer(p))
}

func (e *Engine) getRulepack() *rulepack {
	addr := (*unsafe.Pointer)(unsafe.Pointer(&e.rulepack))
	ptr := atomic.LoadPointer(addr)
	if ptr == nil {
		return nil
	}
	return (*rulepack)(ptr)
}

// PackID returns the ID of the current pack of rules.
func (e *Engine) PackID() (id string) {
	if rulepack := e.getRulepack(); rulepack != nil {
		id = rulepack.id
	}
	return
}

// SetRules set the currents rules. If rules were already set, it will replace
// them by atomically modifying the hooks, and removing what is left.
func (e *Engine) SetRules(packID string, rules []api.Rule) {
	// Create the new rule descriptors and replace the existing ones
	var (
		instrumentationDescriptors hookDescriptorMap
		reactiveRules              []span.EventListener
	)

	// Insert local rules if any
	if localRulesJSON := e.config.LocalRulesFile(); localRulesJSON != "" {
		buf, err := ioutil.ReadFile(localRulesJSON)
		if err == nil {
			var localRules []api.Rule
			err = json.Unmarshal(buf, &localRules)
			if err == nil {
				rules = append(rules, localRules...)
			}
		}

		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, "config: could not read the local rules file"))
		}
	}

	e.logger.Debugf("security rules: loading rules from pack `%s`", packID)
	instrumentationDescriptors, reactiveRules = newRuleDescriptors(e, packID, rules)

	e.setRules(packID, instrumentationDescriptors, reactiveRules)
}

func (e *Engine) setRules(packID string, descriptors hookDescriptorMap, reactiveCallbacks []span.EventListener) {
	// Firstly replace already enabled hookpoints with their new callbacks.
	var (
		currentHooks             hookDescriptorMap
		currentReactiveCallbacks []span.EventListener
	)
	if rulepack := e.getRulepack(); rulepack != nil {
		currentHooks = rulepack.hooks
		currentReactiveCallbacks = rulepack.reactiveCallbacks
	}

	// Create the set of completely disabled hooks while enabling the new ones
	disabledDescriptors := currentHooks
	for hook, descr := range descriptors {
		if e.enabled {
			// Attach the callback to the hook, possibly overwriting the previous one.
			e.logger.Debugf("security rules: attaching callback to `%s`", hook)
			err := hook.Attach(descr.callbacks...)
			if err != nil {
				e.logger.Error(sqerrors.Wrapf(err, "security rules: could not attach the prolog callback to `%s`", hook))
				continue
			}
		}

		// Now close the previously attached callback and then remove it from the
		// set of previous descriptors as it should only contain disabled hooks
		// from previous rules that are no longer present.
		if prevDescr, exists := disabledDescriptors[hook]; exists {
			delete(disabledDescriptors, hook)
			if err := prevDescr.Close(); err != nil {
				e.logger.Error(sqerrors.Wrapf(err, "security rules: error while closing the callback of hook `%v`", hook))
			}
		}
	}

	// Close the previous descriptors that are now completely disabled.
	for hook, descr := range disabledDescriptors {
		e.logger.Debugf("security rules: disabling no longer needed hook `%s`", hook)
		err := hook.Attach(nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: could not disable hook `%v`", hook))
			continue
		}
		if err := descr.Close(); err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: error while closing the callback of hook `%v`", hook))
		}
	}

	// Close the reactive callback objects
	for _, o := range currentReactiveCallbacks {
		if closer, ok := o.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				e.logger.Error(sqerrors.Wrapf(err, "security rules: error while closing the reactive callback"))
			}
		}
	}

	// Set the new rulespack
	e.setRulepack(&rulepack{
		id:                packID,
		hooks:             descriptors,
		reactiveCallbacks: reactiveCallbacks,
	})
}

// newRuleDescriptors walks the list of received rules and creates reactive
// rules and the map of hook descriptors indexed by their hook pointer.
// A hook descriptor contains all it takes to enable and disable rules at run
// time.
func newRuleDescriptors(e *Engine, rulepackID string, rules []api.Rule) (hookDescriptorMap, []span.EventListener) {
	logger := e.logger
	var (
		hookDescriptors = make(hookDescriptorMap)
		reactiveRules   []span.EventListener
	)
	for i := len(rules) - 1; i >= 0; i-- {
		r := &rules[i]
		// Verify the signature
		if err := VerifyRuleSignature(r, e.publicKey); err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: signature verification", r.Name))
			continue
		}

		if r.Reactive != nil || (r.Hookpoint.Callback == "" && r.Hookpoint.Strategy == "") {
			// Reactive engine subscriber rule
			listener, err := newReactiveRule(e, rulepackID, r)
			if err != nil {
				logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: ", r.Name))
				continue
			}
			reactiveRules = append(reactiveRules, listener)
			continue
		}

		// Native instrumentation rule
		hook, prolog, err := newInstrumentationRule(e, rulepackID, r)
		if err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: ", r.Name))
			continue
		}
		e.logger.Debugf("security rules: rule `%s`: successfully found hook `%v`", r.Name, hook)
		hookDescriptors.Add(hook, prolog, r.Priority)
	}

	if rulepackID != "" {
		addStaticRuleDescriptors(logger, e.instrumentationEngine, hookDescriptors, &reactiveRules)
	}

	if len(hookDescriptors) == 0 {
		// No instrumentation rules in the end, return nil instead of an empty map
		hookDescriptors = nil
	}
	return hookDescriptors, reactiveRules
}

func addStaticRuleDescriptors(logger plog.ErrorLogger, instrumentationEngine InstrumentationFace, hookDescriptors hookDescriptorMap, reactiveRules *[]span.EventListener) {
	if staticDescriptors == nil {
		// TODO: wrap the event listener with callback middlewares

		staticDescriptors = &staticRuleDescriptors{
			nativeInstrumentation: make(hookDescriptorMap),
		}

		logRuleErr := func(rule string, err error) {
			logger.Error(sqerrors.Wrapf(err, "security rules: static rule `%s`: ", rule))
		}

		for _, sr := range staticRules {
			switch instrumentation := sr.Instrumentation.(type) {
			default:
				logRuleErr(sr.Name, sqerrors.Errorf("unexpected static rule type `%T`", sr))
				continue

			case NativeInstrumentation:
				hook, err := instrumentationEngine.Find(instrumentation.Function)
				if err != nil {
					logRuleErr(sr.Name, sqerrors.Wrapf(err, "hook lookup of function `%s`", instrumentation.Function))
					continue
				}

				if hook == nil {
					logRuleErr(sr.Name, sqerrors.Errorf("could not find the hook of function `%s`", instrumentation.Function))
					continue
				}

				staticDescriptors.nativeInstrumentation.Add(hook, instrumentation.Callback, instrumentation.Priority)

			case SpanInstrumentation:
				staticDescriptors.spanInstrumentation = append(staticDescriptors.spanInstrumentation, instrumentation.EventListener)
			}
		}
	}

	*reactiveRules = append(*reactiveRules, staticDescriptors.spanInstrumentation...)
	for h, d := range staticDescriptors.nativeInstrumentation {
		for i := range d.callbacks {
			hookDescriptors.Add(h, d.callbacks[i], d.priorities[i])
		}
	}
}

func newReactiveRule(e *Engine, rulepackID string, r *api.Rule) (span.EventListener, error) {
	// Create the rule context
	ruleCtx, err := newNativeRuleContext(r, rulepackID, e.metricsEngine, e.logger, e.perfHistogramUnit, e.perfHistogramBase, e.perfHistogramPeriod)
	if err != nil {
		return nil, sqerrors.Wrap(err, "rule context creation")
	}

	cfg, err := newNativeCallbackConfig(r)
	if err != nil {
		return nil, sqerrors.Wrap(err, "rule callback configuration")
	}

	return newReactiveCallback(r.Hookpoint.Callback, ruleCtx, cfg)
}

func newInstrumentationRule(e *Engine, rulepackID string, r *api.Rule) (HookFace, sqhook.PrologCallback, error) {
	// Find the symbol
	hookpoint := r.Hookpoint
	symbol := hookpoint.Method
	hook, err := e.instrumentationEngine.Find(symbol)
	if err != nil {
		return nil, nil, sqerrors.Wrapf(err, "hook lookup of function `%s`", symbol)
	}

	if hook == nil {
		return nil, nil, sqerrors.Errorf("could not find the hook of function `%s`", symbol)
	}

	// Create the rule context
	ruleCtx, err := newNativeRuleContext(r, rulepackID, e.metricsEngine, e.logger, e.perfHistogramUnit, e.perfHistogramBase, e.perfHistogramPeriod)
	if err != nil {
		return nil, nil, sqerrors.Wrap(err, "rule context creation")
	}

	// Create the prolog callback
	var prolog sqhook.PrologCallback
	switch hookpoint.Strategy {
	case "", "native":
		cfg, err := newNativeCallbackConfig(r)
		if err != nil {
			return nil, nil, sqerrors.Wrap(err, "rule callback configuration")
		}

		prolog, err = NewNativeCallback(hookpoint.Callback, ruleCtx, cfg)
		if err != nil {
			return nil, nil, sqerrors.Wrap(err, "callback constructor")
		}

	case "reflected":
		prolog, err = NewReflectedCallback(hookpoint.Callback, ruleCtx, r)
		if err != nil {
			return nil, nil, sqerrors.Wrap(err, "callback constructor")
		}
	}

	return hook, prolog, nil
}

// Enable the hooks of the ongoing configured rules.
func (e *Engine) Enable() {
	var descriptors hookDescriptorMap
	if rulepack := e.getRulepack(); rulepack != nil {
		descriptors = rulepack.hooks
	}
	for hook, descr := range descriptors {
		e.logger.Debugf("security rules: attaching callback to hook `%s`", hook)
		if err := hook.Attach(descr.callbacks...); err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: could not attach the callback to hook `%v`", hook))
		}
	}
	e.enabled = true
	e.logger.Debugf("security rules: %d security rules enabled", len(descriptors))
}

// Disable the hooks currently attached to callbacks.
func (e *Engine) Disable() {
	e.enabled = false

	var descriptors hookDescriptorMap
	if rulepack := e.getRulepack(); rulepack != nil {
		descriptors = rulepack.hooks
	}

	if l := len(descriptors); l == 0 {
		return
	}
	for hook := range descriptors {
		err := hook.Attach(nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: error while disabling hook `%v`", hook))
		}
	}
	e.logger.Debugf("security rules: %d security rules disabled")
}

func (e *Engine) SpanEventListeners() []span.EventListener {
	var reactiveCallbacks []span.EventListener
	if rulepack := e.getRulepack(); rulepack != nil {
		reactiveCallbacks = rulepack.reactiveCallbacks
	}
	return reactiveCallbacks
}

// Count returns the count of correctly instantiated and enabled rules.
func (e *Engine) Count() (c int) {
	// Not precise but should do the job for now
	if rulepack := e.getRulepack(); rulepack != nil {
		c = len(rulepack.hooks) + len(rulepack.reactiveCallbacks)
	}
	return
}

type (
	hookDescriptorMap map[HookFace]hookDescriptor

	hookDescriptor struct {
		priorities []int
		callbacks  []sqhook.PrologCallback
		closers    []io.Closer
	}
)

func (m hookDescriptorMap) Add(hook HookFace, callback sqhook.PrologCallback, priority int) {
	d, exists := m[hook]
	closer, _ := callback.(io.Closer)

	if !exists {
		// First insertion
		var closers []io.Closer
		if closer != nil {
			closers = []io.Closer{closer}
		}
		m[hook] = hookDescriptor{
			priorities: []int{priority},
			callbacks:  []sqhook.PrologCallback{callback},
			closers:    closers,
		}
		return
	}

	// Not the first insertion.
	// Look for the callback position i per ascending priority order
	i := sort.Search(len(d.priorities), func(i int) bool {
		return d.priorities[i] > priority
	})

	// Update the list of priorities
	d.priorities = append(d.priorities, 0)
	copy(d.priorities[i+1:], d.priorities[i:])
	d.priorities[i] = priority

	// Update the list of closers
	if closer != nil {
		d.closers = append(d.closers, closer)
	}

	// Update the list of callbacks
	d.callbacks = append(d.callbacks, nil)
	copy(d.callbacks[i+1:], d.callbacks[i:])
	d.callbacks[i] = callback

	// Update the hook descriptor map entry with the new value
	m[hook] = d
}

func (d hookDescriptor) Close() error {
	var errs sqerrors.ErrorCollection
	for _, c := range d.closers {
		err := c.Close()
		if err != nil {
			errs.Add(err)
		}
	}
	return errs.ToError()
}

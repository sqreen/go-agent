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
	"io"
	"sort"
	"time"

	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type Engine struct {
	logger Logger
	// Map rules to their corresponding symbol in order to be able to modify them
	// at run time by atomically replacing a running rule.
	// TODO: write a test to check two HookFaces are correctly comparable
	//   to find back a hook
	hooks                                hookDescriptorMap
	packID                               string
	enabled                              bool
	metricsEngine                        *metrics.Engine
	publicKey                            *ecdsa.PublicKey
	instrumentationEngine                InstrumentationFace
	perfHistogramUnit, perfHistogramBase float64
	perfHistogramPeriod                  time.Duration
}

// Logger interface required by this package.
type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

// NewEngine returns a new rule engine.
func NewEngine(logger Logger, instrumentationEngine InstrumentationFace, metricsEngine *metrics.Engine, publicKey *ecdsa.PublicKey, perfHistogramUnit, perfHistogramBase float64, perfHistogramPeriod time.Duration) *Engine {
	if instrumentationEngine == nil {
		instrumentationEngine = defaultInstrumentationEngine
	}

	return &Engine{
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

// PackID returns the ID of the current pack of rules.
func (e *Engine) PackID() string {
	return e.packID
}

// SetRules set the currents rules. If rules were already set, it will replace
// them by atomically modifying the hooks, and removing what is left.
func (e *Engine) SetRules(packID string, rules []api.Rule) {
	// Create the new rule descriptors and replace the existing ones
	var ruleDescriptors hookDescriptorMap
	if len(rules) > 0 {
		e.logger.Debugf("security rules: loading rules from pack `%s`", packID)
		ruleDescriptors = newHookDescriptors(e, packID, rules)
	}
	e.setRules(packID, ruleDescriptors)
}

func (e *Engine) setRules(packID string, descriptors hookDescriptorMap) {
	// Firstly update already enabled hookpoints with their new callbacks in order
	// to avoid having a blank moment without any callback set. This case happens
	// when a rule is updated.
	disabledDescriptors := e.hooks
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

	// Close the previous descriptors that are now disabled.
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

	// Save the rules pack ID and the new set of enabled hooks
	e.packID = packID
	e.hooks = descriptors
}

// newHookDescriptors walks the list of received rules and creates the map of
// hook descriptors indexed by their hook pointer. A hook descriptor contains
// all it takes to enable and disable rules at run time.
func newHookDescriptors(e *Engine, rulepackID string, rules []api.Rule) hookDescriptorMap {
	logger := e.logger

	// Create and configure the list of callbacks according to the given rules
	var hookDescriptors = make(hookDescriptorMap)
	for i := len(rules) - 1; i >= 0; i-- {
		r := rules[i]
		// Verify the signature
		if err := VerifyRuleSignature(&r, e.publicKey); err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: signature verification", r.Name))
			continue
		}
		//Find the symbol
		hookpoint := r.Hookpoint
		symbol := hookpoint.Method
		hook, err := e.instrumentationEngine.Find(symbol)
		if err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: unexpected error while looking for the hook of `%s`", r.Name, symbol))
			continue
		}
		if hook == nil {
			logger.Debugf("security rules: rule `%s`: could not find the hook of function `%s`", r.Name, symbol)
			continue
		} else {
			logger.Debugf("security rules: rule `%s`: successfully found hook `%v`", r.Name, hook)
		}

		// Create the rule context
		ruleCtx, err := newNativeRuleContext(&r, rulepackID, e.metricsEngine, e.logger, e.perfHistogramUnit, e.perfHistogramBase, e.perfHistogramPeriod)
		if err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: callback configuration", r.Name))
			continue
		}

		// Create the prolog callback
		var prolog sqhook.PrologCallback
		switch hookpoint.Strategy {
		case "", "native":
			cfg, err := newNativeCallbackConfig(&r)
			if err != nil {
				logger.Error(sqerrors.Wrap(err, "callback configuration"))
				continue
			}

			prolog, err = NewNativeCallback(hookpoint.Callback, ruleCtx, cfg)
			if err != nil {
				logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: callback constructor", r.Name))
				continue
			}

		case "reflected":
			prolog, err = NewReflectedCallback(hookpoint.Callback, ruleCtx, &r)
			if err != nil {
				logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: callback constructor", r.Name))
				continue
			}
		}

		// Create the descriptor with everything required to be able to enable or
		// disable it afterwards.
		hookDescriptors.Add(hook, prolog, r.Priority)
	}
	// Nothing in the end
	if len(hookDescriptors) == 0 {
		return nil
	}
	return hookDescriptors
}

// Enable the hooks of the ongoing configured rules.
func (e *Engine) Enable() {
	for hook, descr := range e.hooks {
		e.logger.Debugf("security rules: attaching callback to hook `%s`", hook)
		if err := hook.Attach(descr.callbacks...); err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: could not attach the callback to hook `%v`", hook))
		}
	}
	e.enabled = true
	e.logger.Debugf("security rules: %d security rules enabled", len(e.hooks))
}

// Disable the hooks currently attached to callbacks.
func (e *Engine) Disable() {
	e.enabled = false
	for hook := range e.hooks {
		err := hook.Attach(nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrapf(err, "security rules: error while disabling hook `%v`", hook))
		}
	}
	e.logger.Debugf("security rules: %d security rules disabled", len(e.hooks))
}

// Count returns the count of correctly instantiated and enabled rules.
func (e *Engine) Count() int {
	// Not precise but should do the job for now
	return len(e.hooks)
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

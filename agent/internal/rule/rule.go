// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

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
	"fmt"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/metrics"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

type Engine struct {
	logger Logger
	// Map rules to their corresponding symbol in order to be able to modify them
	// at run time by atomically replacing a running rule.
	hooks         hookDescriptors
	packID        string
	cfg           *config.Config
	enabled       bool
	metricsEngine *metrics.Engine
	publicKey     *ecdsa.PublicKey
}

// Logger interface required by this package.
type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

// NewEngine returns a new rule engine.
func NewEngine(logger Logger, metricsEngine *metrics.Engine, publicKey *ecdsa.PublicKey) *Engine {
	return &Engine{
		logger:        logger,
		metricsEngine: metricsEngine,
		publicKey:     publicKey,
	}
}

// PackID returns the ID of the current pack of rules.
func (e *Engine) PackID() string {
	return e.packID
}

// SetRules set the currents rules. If rules were already set, it will replace
// them by atomically modifying the hooks, and removing what is left.
func (e *Engine) SetRules(packID string, rules []api.Rule, errorMetricsStore *metrics.Store) {
	// Create the net rule descriptors and replace the existing ones
	ruleDescriptors := newHookDescriptors(e.logger, rules, e.publicKey, e.metricsEngine, errorMetricsStore)
	e.setRules(packID, ruleDescriptors)
}

func (e *Engine) setRules(packID string, descriptors hookDescriptors) {
	// Firstly update already enabled hookpoints with their new callbacks in order
	// to avoid having a blank without any callback. This case happens when a rule
	// is updated.
	disabledDescriptors := e.hooks
	for hook, descr := range descriptors {
		if e.enabled {
			// Attach the callback to the hook, possibly overwriting the previous one.
			err := hook.Attach(descr.Prolog())
			if err != nil {
				e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: could not attach the callbacks")))
				continue
			}
		}

		// Now close the previously attached callback and then remove it from the
		// set of previous descriptors as it should now only contain disabled hook
		// points from previous rules that are no longer present.
		if prevDescr, exists := disabledDescriptors[hook]; exists {
			delete(disabledDescriptors, hook)
			if err := prevDescr.Close(); err != nil {
				e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: error while closing the callback of hook `%v`", hook)))
			}
		}
	}

	// Close the previous descriptors that are now disabled.
	for hook, descr := range disabledDescriptors {
		err := hook.Attach(nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: could not attach the callbacks")))
			continue
		}
		if err := descr.Close(); err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: error while closing the callback of hook `%v`", hook)))
		}
	}

	// Save the rules pack ID and the new set of enabled hooks
	e.packID = packID
	e.hooks = descriptors
}

// newHookDescriptors walks the list of received rules and creates the map of
// hook descriptors indexed by their hook pointer. A hook descriptor contains
// all it takes to enable and disable rules at run time.
func newHookDescriptors(logger Logger, rules []api.Rule, publicKey *ecdsa.PublicKey, metricsEngine *metrics.Engine, errorMetricsStore *metrics.Store) hookDescriptors {
	// Create and configure the list of callbacks according to the given rules
	var hookDescriptors = make(hookDescriptors)
	for i := len(rules) - 1; i >= 0; i-- {
		r := rules[i]
		// Verify the signature
		if err := VerifyRuleSignature(&r, publicKey); err != nil {
			logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: signature verification", r.Name)))
			continue
		}
		// Find the symbol
		hookpoint := r.Hookpoint
		symbol := fmt.Sprintf("%s.%s", hookpoint.Class, hookpoint.Method)
		hook := sqhook.Find(symbol)
		if hook == nil {
			logger.Debugf("rule `%s` ignored: symbol `%s` cannot be hooked", r.Name, symbol)
			continue
		}
		// Instantiate the callback
		descr := hookDescriptors.Get(hook)
		var nextProlog sqhook.PrologCallback
		if descr != nil {
			nextProlog = descr.Prolog()
		}
		ruleDescriptor := NewCallbackContext(&r, logger, metricsEngine, errorMetricsStore)
		prolog, err := NewCallback(hookpoint.Callback, ruleDescriptor, nextProlog)
		if err != nil {
			logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not instantiate the callbacks", r.Name)))
			continue
		}
		// Create the descriptor with everything required to be able to enable or
		// disable it afterwards.
		hookDescriptors.Set(hook, prolog)
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
		err := hook.Attach(descr.Prolog())
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: could not attach the callback `%v` to hook `%v`", descr.Prolog(), hook)))
		}
	}
	e.enabled = true
}

// Disable the hooks currently attached to callbacks.
func (e *Engine) Disable() {
	for hook := range e.hooks {
		err := hook.Attach(nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule: could not disable hook `%v`", hook)))
		}
	}
	e.enabled = false
}

type hookDescriptors map[*sqhook.Hook]CallbackObject

type noopCloserCallbackObject struct {
	prolog sqhook.PrologCallback
}

func (cbo noopCloserCallbackObject) Prolog() sqhook.PrologCallback {
	return cbo.prolog
}

func (cbo noopCloserCallbackObject) Close() error {
	return nil
}

// chainedCallbackObject is the set of callbacks attached to a hookpoint.
// Callbacks already have a reference to the next callback, and it is their
// responsibility to call it. But closing them is done agent-side using this
// chain.
// FIXME: better design to simplify and unify this
type chainedCallbackObject struct {
	current, next CallbackObject
}

func (c *chainedCallbackObject) Prolog() sqhook.PrologCallback {
	return c.current.Prolog()
}

func (c *chainedCallbackObject) Close() error {
	err1 := c.current.Close()
	var err2 error
	if c.next != nil {
		err2 = c.next.Close()
	}
	// FIXME: create a "error set" error type
	if err1 != nil {
		return err1
	}
	return err2
}

func (m hookDescriptors) Set(hook *sqhook.Hook, cb interface{}) {
	var callback CallbackObject
	if cbo, ok := cb.(CallbackObject); ok {
		callback = cbo
	} else {
		callback = noopCloserCallbackObject{cb}
	}
	current := m[hook]
	m[hook] = &chainedCallbackObject{current: callback, next: current}
}

func (m hookDescriptors) Get(hook *sqhook.Hook) CallbackObject {
	return m[hook]
}

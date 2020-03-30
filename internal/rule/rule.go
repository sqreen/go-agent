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
	"fmt"
	"io"

	"github.com/sqreen/go-agent/internal/app"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/metrics"
	"github.com/sqreen/go-agent/internal/plog"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

type Engine struct {
	logger Logger
	// Map rules to their corresponding symbol in order to be able to modify them
	// at run time by atomically replacing a running rule.
	// TODO: write a test to check two HookFaces are correctly comparable
	//   to find back a hook
	hooks                 hookDescriptors
	packID                string
	enabled               bool
	metricsEngine         *metrics.Engine
	publicKey             *ecdsa.PublicKey
	vendorPrefix          string
	instrumentationEngine InstrumentationFace
	errorMetricsStore     *metrics.Store
}

// Logger interface required by this package.
type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

// NewEngine returns a new rule engine.
func NewEngine(logger Logger, instrumentationEngine InstrumentationFace, metricsEngine *metrics.Engine, errorMetricsStore *metrics.Store, publicKey *ecdsa.PublicKey) *Engine {
	if instrumentationEngine == nil {
		instrumentationEngine = defaultInstrumentationEngine
	}
	return &Engine{
		logger:                logger,
		metricsEngine:         metricsEngine,
		publicKey:             publicKey,
		vendorPrefix:          app.VendorPrefix(),
		instrumentationEngine: instrumentationEngine,
		errorMetricsStore:     errorMetricsStore,
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
	var ruleDescriptors hookDescriptors
	if len(rules) > 0 {
		e.logger.Debugf("security rules: loading rules from pack `%s`", packID)
		ruleDescriptors = newHookDescriptors(e, rules)
	}
	e.setRules(packID, ruleDescriptors)
}

func (e *Engine) setRules(packID string, descriptors hookDescriptors) {
	// Firstly update already enabled hookpoints with their new callbacks in order
	// to avoid having a blank moment without any callback set. This case happens
	// when a rule is updated.
	disabledDescriptors := e.hooks
	for hook, descr := range descriptors {
		if e.enabled {
			// Attach the callback to the hook, possibly overwriting the previous one.
			e.logger.Debugf("security rules: attaching callback to `%s`", hook)
			err := hook.Attach(descr.callback)
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
func newHookDescriptors(e *Engine, rules []api.Rule) hookDescriptors {
	logger := e.logger

	// Create and configure the list of callbacks according to the given rules
	var hookDescriptors = make(hookDescriptors)
	for i := len(rules) - 1; i >= 0; i-- {
		r := rules[i]
		// Verify the signature
		if err := VerifyRuleSignature(&r, e.publicKey); err != nil {
			// TODO: implement for reflected callbacks
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
		if hook == nil && e.vendorPrefix != "" {
			hook, err = e.instrumentationEngine.Find(e.vendorPrefix + symbol)
			if err != nil {
				logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: unexpected error while looking for the hook of `%s`", r.Name, symbol))
				continue
			}
		}
		if hook == nil {
			logger.Debugf("security rules: rule `%s`: could not find the hook of function `%s`", r.Name, symbol)
			continue
		} else {
			logger.Debugf("security rules: rule `%s`: successfully found hook `%v`", r.Name, hook)
		}

		// Instantiate the callback
		//descr := hookDescriptors.Get(hook)
		//var nextProlog sqhook.PrologCallback
		//if next := descr.callback; next != nil {
		//	nextProlog = next
		//}

		callbackContext, err := NewCallbackContext(&r, e.metricsEngine, e.errorMetricsStore)
		if err != nil {
			logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: callback configuration", r.Name))
			continue
		}

		switch hookpoint.Strategy {
		case "":
			fallthrough
		case "native":
			prolog, err := callback.NewNativeCallback(hookpoint.Callback, callbackContext)
			if err != nil {
				logger.Error(sqerrors.Wrapf(err, "security rules: rule `%s`: callback constructor", r.Name))
				continue
			}
			// Create the descriptor with everything required to be able to enable or
			// disable it afterwards.
			hookDescriptors.Set(hook, prolog)

		case "reflected":
			prolog, err := callback.NewReflectedCallback(hookpoint.Callback, hook.PrologFuncType(), callbackContext)
			if err != nil {
				logger.Error(sqerrors.Wrap(err, fmt.Sprintf("security rules: rule `%s`: callback constructor", r.Name)))
				continue
			}
			// Create the descriptor with everything required to be able to enable or
			// disable it afterwards.
			hookDescriptors.Set(hook, prolog)
		}

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
		prolog := descr.callback
		e.logger.Debugf("security rules: attaching callback to hook `%s`", hook)
		if err := hook.Attach(prolog); err != nil {
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

type callbackWrapper struct {
	callback sqhook.PrologCallback
}

func (c callbackWrapper) Close() error {
	if closer, ok := c.callback.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

//type hookDescriptors map[HookFace]CallbackObject

//type noopCloserCallbackObject struct {
//	prolog sqhook.PrologCallback
//}
//
//func (cbo noopCloserCallbackObject) Prolog() sqhook.PrologCallback {
//	return cbo.prolog
//}
//
//func (cbo noopCloserCallbackObject) Close() error {
//	return nil
//}

// chainedCallbackObject is the set of callbacks attached to a hookpoint.
// Callbacks already have a reference to the next callback, and it is their
// responsibility to call it. But closing them is done agent-side using this
// chain.
// FIXME: better design to simplify and unify this
//type chainedCallbackObject struct {
//	current, next CallbackObject
//}

//func (c *chainedCallbackObject) Prolog() sqhook.PrologCallback {
//	return c.current.Prolog()
//}
//
//func (c *chainedCallbackObject) Close() error {
//	err1 := c.current.Close()
//	var err2 error
//	if c.next != nil {
//		err2 = c.next.Close()
//	}
//	// FIXME: create a "error set" error type
//	if err1 != nil {
//		return err1
//	}
//	return err2
//}

//
//func (m hookDescriptors) Set(hook HookFace, prolog interface{}) {
//	var callback CallbackObject
//	if cbo, ok := prolog.(CallbackObject); ok {
//		callback = cbo
//	} else {
//		callback = noopCloserCallbackObject{prolog}
//	}
//	current := m[hook]
//	m[hook] = &chainedCallbackObject{current: callback, next: current}
//}

type hookDescriptors map[HookFace]callbackWrapper

func (m hookDescriptors) Set(hook HookFace, prolog sqhook.PrologCallback) {
	m[hook] = callbackWrapper{prolog}
}

func (m hookDescriptors) Get(hook HookFace) callbackWrapper {
	return m[hook]
}

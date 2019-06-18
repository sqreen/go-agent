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
	"fmt"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
	"github.com/sqreen/go-agent/agent/internal/plog"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

type Engine struct {
	logger Logger
	// Map rules to their corresponding symbol in order to be able to modify them
	// at run time by atomically replacing a running rule.
	rules   ruleDescriptors
	packID  string
	cfg     *config.Config
	enabled bool
}

// Logger interface required by this package.
type Logger interface {
	plog.DebugLogger
	plog.ErrorLogger
}

// NewEngine returns a new rule engine.
func NewEngine(logger Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// PackID returns the ID of the current pack of rules.
func (e *Engine) PackID() string {
	return e.packID
}

// SetRules set the currents rules. If rules were already set, it will replace
// them by atomically modifying the hooks, and removing what is left.
func (e *Engine) SetRules(packID string, rules []api.Rule) {
	// Create the net rule descriptors and replace the existing ones
	ruleDescriptors := newRuleDescriptors(e.logger, rules)
	e.setRules(packID, ruleDescriptors)
}

func (e *Engine) setRules(packID string, descriptors ruleDescriptors) {
	for symbol, rule := range descriptors {
		if e.enabled {
			// TODO: chain multiple callbacks per hookpoint using a callback of callbacks
			//       Attach the callback to the hook
			err := rule.hook.Attach(rule.prolog, rule.epilog)
			if err != nil {
				e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not attach the callbacks", rule.name)))
				continue
			}
		}
		// Remove from the previous rules pack the entries that were redefined in
		// this one.
		delete(e.rules, symbol)
	}
	// Disable previously enabled rules that were not replaced by new ones.
	for _, rule := range e.rules {
		err := rule.hook.Attach(nil, nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not attach the callbacks", rule.name)))
			continue
		}
	}
	// Save the rules pack ID and the list of enabled hooks
	e.packID = packID
	e.rules = descriptors
}

// newRuleDescriptors walks the list of received rules and creates the map of
// rule descriptors indexed by their symbol. A rule descriptor contains all it
// needs to enable and disable rules at run time.
func newRuleDescriptors(logger Logger, rules []api.Rule) ruleDescriptors {
	// Create and configure the list of callbacks according to the given rules
	ruleDescriptors := make(ruleDescriptors)
	for _, r := range rules {
		hookpoint := r.Hookpoint
		// Find the symbol
		symbol := fmt.Sprintf("%s.%s", hookpoint.Class, hookpoint.Method)
		hook := sqhook.Find(symbol)
		if hook == nil {
			logger.Debugf("rule `%s` ignored: symbol `%s` cannot be hooked", r.Name, symbol)
			continue
		}
		// Get the callback data from the API message
		var data []interface{}
		if nbData := len(r.Data.Values); nbData > 0 {
			data = make([]interface{}, 0, nbData)
			for _, e := range r.Data.Values {
				data = append(data, e.Value)
			}
		}
		// Instantiate the callback
		prolog, epilog, err := NewCallbacks(hookpoint.Callback, data)
		if err != nil {
			logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not instantiate the callbacks", r.Name)))
			continue
		}
		// Create the rule descriptor with everything required to be able to enable
		// or disable it afterwards.
		ruleDescriptors.Add(symbol, ruleDescriptor{
			name:   r.Name,
			hook:   hook,
			prolog: prolog,
			epilog: epilog,
		})
	}
	if len(ruleDescriptors) == 0 {
		return nil
	}
	return ruleDescriptors
}

// Enable the hooks of the ongoing configured rules.
func (e *Engine) Enable() {
	for _, r := range e.rules {
		err := r.hook.Attach(r.prolog, r.epilog)
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not attach the callbacks", r.name)))
		}
	}
	e.enabled = true
}

// Disable the hooks currently attached to callbacks.
func (e *Engine) Disable() {
	for _, r := range e.rules {
		err := r.hook.Attach(nil, nil)
		if err != nil {
			e.logger.Error(sqerrors.Wrap(err, fmt.Sprintf("rule `%s`: could not disable the callbacks", r.name)))
		}
	}
	e.enabled = false
}

type ruleDescriptors map[string]ruleDescriptor

type ruleDescriptor struct {
	name           string
	hook           *sqhook.Hook
	epilog, prolog sqhook.Callback
}

func (m ruleDescriptors) Add(symbol string, descriptor ruleDescriptor) {
	m[symbol] = descriptor
}

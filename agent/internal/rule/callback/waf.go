// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/binding-accessor"
	"github.com/sqreen/go-agent/agent/internal/httphandler"
	"github.com/sqreen/go-agent/agent/internal/record"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
	"github.com/sqreen/go-libsqreen/waf"
	waf_types "github.com/sqreen/go-libsqreen/waf/types"
)

const defaultMaxWAFTimeBudget = 3 * time.Millisecond

func NewWAFCallback(rule Context, nextProlog sqhook.PrologCallback) (callback interface{}, err error) {
	cfg, ok := rule.Config().(*api.WAFRuleDataEntry)
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `*api.WAFRuleDataEntry`", cfg)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, sqerrors.New("could not generate a uuid")
	}

	wafRule, err := waf.NewRule(id.String(), cfg.WAFRules)
	if err != nil {
		return nil, sqerrors.New("could not instantiate the in-app waf rule")
	}

	if len(cfg.BindingAccessors) == 0 {
		return nil, sqerrors.New("unexpected empty list of binding accessors")
	}
	bindingAccessors := make(map[string]bindingaccessor.BindingAccessorFunc, len(cfg.BindingAccessors))
	for _, expr := range cfg.BindingAccessors {
		ba, err := bindingaccessor.Compile(expr)
		if err != nil {
			return nil, sqerrors.Wrap(err, fmt.Sprintf("could not compile binding accessor expression `%s`", expr))
		}
		bindingAccessors[expr] = ba
	}

	// Next callbacks to call
	actualNextProlog, ok := nextProlog.(WAFPrologCallbackType)
	if nextProlog != nil && !ok {
		return nil, sqerrors.Errorf("unexpected next prolog type `%T`", nextProlog)
	}

	var timeout time.Duration
	if cfg.Timeout != 0 {
		timeout = time.Duration(cfg.Timeout) * time.Millisecond
	} else {
		timeout = defaultMaxWAFTimeBudget
	}

	return newWAFPrologCallback(rule, wafRule, bindingAccessors, timeout, actualNextProlog), nil
}

type WAFPrologCallbackType = func(*http.ResponseWriter, **http.Request) (WAFEpilogCallbackType, error)
type WAFEpilogCallbackType = func(*error)

var WAFProtectionError = errors.New("waf protection triggered")

func newWAFPrologCallback(ctx Context, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration, next WAFPrologCallbackType) *wafCallbackObject {
	return &wafCallbackObject{
		wafRule: wafRule,
		prolog: func(w *http.ResponseWriter, r **http.Request) (WAFEpilogCallbackType, error) {
			req := *r
			rr := record.FromContext(req.Context())
			baCtx := bindingaccessor.MakeBindingAccessorContext(req, rr.ClientIP().String())
			args := make(waf_types.DataSet, len(bindingAccessors))
			for expr, ba := range bindingAccessors {
				value, err := ba(baCtx)
				if err != nil {
					// Log the error and continue
					ctx.Error(sqerrors.Wrapf(err, "binding accessor execution error `%s`", expr))
					continue
				}
				if value == nil {
					// Skip unset values
					continue
				}
				args[expr] = value
			}

			action, info, err := wafRule.Run(args, timeout)
			if err != nil {
				ctx.Error(sqerrors.Wrap(newWAFRunError(err, args, timeout), "waf rule execution error"))
			} else if err == waf_types.ErrTimeout {
				// no-op: we don't log on the hot path unless an error occurred
			} else {
				info := api.WAFAttackInfo{WAFData: string(info)}
				if ctx.BlockingMode() && action == waf_types.BlockAction {
					// Write the blocking response.
					httphandler.WriteResponse(*w, *r, nil, http.StatusBadRequest, nil)
					// Report the event
					rr.AddAttackEvent(ctx.NewAttack(true, info))
					// Return the epilog and abort the call.
					return func(err *error) {
						// An error needs to be written in order to abort handling the
						// request.
						*err = WAFProtectionError
					}, sqhook.AbortError
				} else if action == waf_types.BlockAction || action == waf_types.MonitorAction {
					// Report the event
					rr.AddAttackEvent(ctx.NewAttack(false, info))
				}
			}

			if next == nil {
				return nil, nil
			}
			return next(w, r)
		},
	}
}

type wafRunErrorInfo struct {
	Input   waf_types.DataSet
	Timeout time.Duration
}

func newWAFRunError(err error, args waf_types.DataSet, timeout time.Duration) error {
	return sqerrors.WithInfo(err, wafRunErrorInfo{
		Input:   args,
		Timeout: timeout,
	})
}

type wafCallbackObject struct {
	wafRule waf_types.Rule
	prolog  WAFPrologCallbackType
}

func (o *wafCallbackObject) Prolog() sqhook.PrologCallback {
	return o.prolog
}

func (o *wafCallbackObject) Close() error {
	return o.wafRule.Close()
}

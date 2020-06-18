// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sqreen/go-agent/internal/backend/api"
	"github.com/sqreen/go-agent/internal/binding-accessor"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	"github.com/sqreen/go-libsqreen/waf"
	waf_types "github.com/sqreen/go-libsqreen/waf/types"
)

const defaultMaxWAFTimeBudget = 3 * time.Millisecond

func NewWAFCallback(rule RuleFace, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	data, ok := cfg.Data().(*api.WAFRuleDataEntry)
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `*api.WAFRuleDataEntry`", data)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, sqerrors.New("could not generate a uuid")
	}

	wafRule, err := waf.NewRule(id.String(), data.WAFRules, bindingaccessor.NewValueMaxElements, bindingaccessor.MaxExecutionDepth)
	if err != nil {
		return nil, sqerrors.Wrap(err, "could not instantiate the in-app waf rule")
	}

	if len(data.BindingAccessors) == 0 {
		return nil, sqerrors.New("unexpected empty list of binding accessors")
	}
	bindingAccessors := make(map[string]bindingaccessor.BindingAccessorFunc, len(data.BindingAccessors))
	for _, expr := range data.BindingAccessors {
		ba, err := bindingaccessor.Compile(expr)
		if err != nil {
			return nil, sqerrors.Wrap(err, fmt.Sprintf("could not compile binding accessor expression `%s`", expr))
		}
		bindingAccessors[expr] = ba
	}

	var timeout time.Duration
	if data.Timeout != 0 {
		timeout = time.Duration(data.Timeout) * time.Millisecond
	} else {
		timeout = defaultMaxWAFTimeBudget
	}

	return newWAFPrologCallback(rule, cfg.BlockingMode(), wafRule, bindingAccessors, timeout), nil
}

type (
	WAFPrologCallbackType = httpprotection.BlockingPrologCallbackType
	WAFEpilogCallbackType = httpprotection.BlockingEpilogCallbackType
)

var WAFProtectionError = errors.New("waf protection triggered")

func newWAFPrologCallback(rule RuleFace, blockingMode bool, wafRule waf_types.Rule, bindingAccessors map[string]bindingaccessor.BindingAccessorFunc, timeout time.Duration) *wafCallbackObject {
	return &wafCallbackObject{
		wafRule: wafRule,
		prolog: func(p **httpprotection.RequestContext) (httpprotection.BlockingEpilogCallbackType, error) {
			ctx := *p
			baCtx := MakeWAFCallbackBindingAccessorContext(ctx.RequestReader)
			args := make(waf_types.DataSet, len(bindingAccessors))
			for expr, ba := range bindingAccessors {
				value, err := ba(baCtx)
				if err != nil {
					// Log the error and continue
					ctx.Logger().Error(sqerrors.Wrapf(err, "binding accessor execution error `%s`", expr))
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
				ctx.Logger().Error(sqerrors.Wrap(newWAFRunError(err, args, timeout), "waf rule execution error"))
			} else if err == waf_types.ErrTimeout {
				// no-op: we don't log in such a hot path
			} else {
				info := api.WAFAttackInfo{WAFData: string(info)}
				if blockingMode && action == waf_types.BlockAction {
					// Report the event
					ctx.AddAttackEvent(rule.NewAttackEvent(true, info, nil))
					ctx.WriteDefaultBlockingResponse()
					// Return the epilog and abort the call.
					return func(err *error) {
						// An error needs to be written in order to abort handling the
						// request.
						*err = WAFProtectionError
					}, sqhook.AbortError
				} else if action == waf_types.BlockAction || action == waf_types.MonitorAction {
					// Report the event
					ctx.AddAttackEvent(rule.NewAttackEvent(false, info, nil))
				}
			}

			return nil, nil
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

func (o *wafCallbackObject) PrologCallback() sqhook.PrologCallback {
	return o.prolog
}

func (o *wafCallbackObject) Close() error {
	return o.wafRule.Close()
}

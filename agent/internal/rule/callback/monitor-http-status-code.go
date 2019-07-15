// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

func NewMonitorHTTPStatusCodeCallbacks(rule Context, nextProlog, nextEpilog sqhook.Callback) (prolog, epilog sqhook.Callback, err error) {
	// Next callbacks to call
	actualNextProlog, ok := nextProlog.(MonitorHTTPStatusCodePrologCallbackType)
	if nextProlog != nil && !ok {
		err = sqerrors.Errorf("unexpected next prolog type `%T` instead of `%T`", nextProlog, MonitorHTTPStatusCodePrologCallbackType(nil))
		return
	}
	// No epilog in this callback, so simply check and pass the given one
	if _, ok := nextEpilog.(MonitorHTTPStatusCodeEpilogCallbackType); nextEpilog != nil && !ok {
		err = sqerrors.Errorf("unexpected next epilog type `%T` instead of `%T`", nextEpilog, MonitorHTTPStatusCodeEpilogCallbackType(nil))
		return
	}
	return newMonitorHTTPStatusCodePrologCallback(rule, actualNextProlog), nextEpilog, nil
}

func newMonitorHTTPStatusCodePrologCallback(rule Context, next MonitorHTTPStatusCodePrologCallbackType) MonitorHTTPStatusCodePrologCallbackType {
	return func(ctx *sqhook.Context, code *int) error {
		//if status := *code; status >= 400 && status <= 500 {
		rule.AddMetricsValue(*code, 1)
		//}

		if next == nil {
			return nil
		}
		return next(ctx, code)
	}
}

type MonitorHTTPStatusCodePrologCallbackType = func(*sqhook.Context, *int) error
type MonitorHTTPStatusCodeEpilogCallbackType = func(*sqhook.Context)

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

func NewMonitorHTTPStatusCodeCallbacks(rule Context, nextProlog sqhook.PrologCallback) (prolog sqhook.PrologCallback, err error) {
	// Next callbacks to call
	actualNextProlog, ok := nextProlog.(MonitorHTTPStatusCodePrologCallbackType)
	if nextProlog != nil && !ok {
		err = sqerrors.Errorf("unexpected next prolog type `%T` instead of `%T`", nextProlog, MonitorHTTPStatusCodePrologCallbackType(nil))
		return
	}
	return newMonitorHTTPStatusCodePrologCallback(rule, actualNextProlog), nil
}

func newMonitorHTTPStatusCodePrologCallback(rule Context, next MonitorHTTPStatusCodePrologCallbackType) MonitorHTTPStatusCodePrologCallbackType {
	return func(r sqhook.MethodReceiver, code *int) (MonitorHTTPStatusCodeEpilogCallbackType, error) {
		var (
			nextEpilog MonitorHTTPStatusCodeEpilogCallbackType
			err        error
		)
		if next != nil {
			nextEpilog, err = next(r, code)
		}
		return newMonitorHTTPStatusCodeEpilogCallback(rule, code, nextEpilog), err
	}
}

func newMonitorHTTPStatusCodeEpilogCallback(rule Context, code *int, next MonitorHTTPStatusCodeEpilogCallbackType) MonitorHTTPStatusCodeEpilogCallbackType {
	return func() {
		if next != nil {
			defer next()
		}
		rule.PushMetricsValue(*code, 1)
	}
}

type MonitorHTTPStatusCodeEpilogCallbackType = func()
type MonitorHTTPStatusCodePrologCallbackType = func(sqhook.MethodReceiver, *int) (MonitorHTTPStatusCodeEpilogCallbackType, error)

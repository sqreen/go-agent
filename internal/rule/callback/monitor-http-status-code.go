// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

func NewMonitorHTTPStatusCodeCallback(r RuleContext, _ NativeCallbackConfig) (sqhook.PrologCallback, error) {
	return newMonitorHTTPStatusCodePrologCallback(r), nil
}

func newMonitorHTTPStatusCodePrologCallback(r RuleContext) http_protection.ResponseMonitoringPrologCallbackType {
	return func(_ **http_protection.ProtectionContext, resp *types.ResponseFace) (http_protection.NonBlockingEpilogCallbackType, error) {
		r.Pre(func(c CallbackContext) {
			_ = c.PushMetricsValue((*resp).Status(), 1)
		})
		return nil, nil
	}
}

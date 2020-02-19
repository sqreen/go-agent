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

func NewMonitorHTTPStatusCodeCallback(rule RuleFace) (sqhook.PrologCallback, error) {
	return newMonitorHTTPStatusCodePrologCallback(rule), nil
}

func newMonitorHTTPStatusCodePrologCallback(rule RuleFace) http_protection.ResponseMonitoringPrologCallbackType {
	return func(_ **http_protection.RequestContext, r *types.ResponseFace) (http_protection.NonBlockingEpilogCallbackType, error) {
		_ = rule.PushMetricsValue((*r).Status(), 1)
		return nil, nil
	}
}

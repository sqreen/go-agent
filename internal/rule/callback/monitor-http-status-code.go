// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"net/http"

	"github.com/sqreen/go-agent/internal/event"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

func NewMonitorHTTPStatusCodeCallback(r RuleContext, _ NativeCallbackConfig) (sqhook.PrologCallback, error) {
	return newMonitorHTTPStatusCodePrologCallback(r), nil
}

func newMonitorHTTPStatusCodePrologCallback(r RuleContext) http_protection.ResponseMonitoringPrologCallbackType {
	return func(_ **http_protection.ProtectionContext, resp *types.ResponseFace) (http_protection.NonBlockingEpilogCallbackType, error) {
		r.Pre(func(c CallbackContext) error {
			sqassert.NotNil(resp)
			status := (*resp).Status()
			_ = c.AddMetricsValue(status, 1)
			if status == http.StatusNotFound {
				// Enforce test to true despite the rule's - current backend-internals
				// detail
				blocked := c.HandleAttack(false, event.WithAttackInfo(struct{}{}), event.WithTest(true))
				sqassert.False(blocked)
			}
			return nil
		})
		return nil, nil
	}
}

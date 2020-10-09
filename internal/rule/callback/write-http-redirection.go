// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"net/http"
	"net/url"

	"github.com/sqreen/go-agent/internal/backend/api"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// NewWriteHTTPRedirectionCallbacks returns the native callback applying the
// the rule-configured HTTP redirection to the HTTP protection response writer
// using the URL provided by the rule's data.
func NewWriteHTTPRedirectionCallbacks(_ NativeRuleContext, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	var redirectionURL string
	if cfg := cfg.Data(); cfg != nil {
		cfg, ok := cfg.(*api.RedirectionRuleDataEntry)
		if !ok {
			return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `*api.CustomErrorPageRuleDataEntry`", cfg)
		}
		redirectionURL = cfg.RedirectionURL
	}
	if redirectionURL == "" {
		return nil, sqerrors.New("unexpected empty redirection url")
	}
	if _, err := url.ParseRequestURI(redirectionURL); err != nil {
		return nil, sqerrors.Wrap(err, "validation error of the redirection url")
	}

	return newWriteHTTPRedirectionPrologCallback(redirectionURL), nil
}

// The prolog callback modifies the function arguments in order to perform an
// HTTP redirection.
func newWriteHTTPRedirectionPrologCallback(url string) httpprotection.NonBlockingPrologCallbackType {
	return func(m **httpprotection.ProtectionContext) (httpprotection.NonBlockingEpilogCallbackType, error) {
		ctx := *m
		ctx.ResponseWriter.Header().Set("Location", url)
		ctx.ResponseWriter.WriteHeader(http.StatusSeeOther)
		return nil, nil
	}
}

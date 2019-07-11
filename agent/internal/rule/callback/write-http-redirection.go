// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"net/http"
	"net/url"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// NewWriteHTTPRedirectionCallbacks returns the native prolog and epilog
// callbacks modifying the arguments of `httphandler.WriteResponse` in order to
// modify the http status code and headers in order to perform an HTTP
// redirection to the URL provided by the rule's data.
func NewWriteHTTPRedirectionCallbacks(data []interface{}) (prolog, epilog sqhook.Callback, err error) {
	var redirectionURL string
	if len(data) > 0 {
		d0 := data[0]
		cfg, ok := d0.(*api.RedirectionRuleDataEntry)
		if !ok {
			return nil, nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `*api.CustomErrorPageRuleDataEntry`", d0)
		}
		redirectionURL = cfg.RedirectionURL
	}
	if redirectionURL == "" {
		return nil, nil, sqerrors.New("unexpected empty redirection url")
	}
	if _, err := url.ParseRequestURI(redirectionURL); err != nil {
		return nil, nil, sqerrors.Wrap(err, "validation error of the redirection url")
	}
	return newWriteHTTPRedirectionPrologCallback(redirectionURL), nil, nil
}

type WriteHTTPRedirectionPrologCallbackType = func(*sqhook.Context, *http.ResponseWriter, **http.Request, *http.Header, *int, *[]byte) error

// The prolog callback modifies the function arguments in order to perform an
// HTTP redirection.
func newWriteHTTPRedirectionPrologCallback(url string) WriteHTTPRedirectionPrologCallbackType {
	return func(_ *sqhook.Context, _ *http.ResponseWriter, _ **http.Request, callerHeaders *http.Header, callerStatusCode *int, _ *[]byte) error {
		*callerStatusCode = http.StatusSeeOther
		if *callerHeaders == nil {
			*callerHeaders = make(http.Header)
		}
		callerHeaders.Set("Location", url)
		return nil
	}
}

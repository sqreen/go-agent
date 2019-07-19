// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// NewAddSecurityHeadersCallbacks returns the native prolog and epilog callbacks
// to be hooked to `sqhttp.MiddlewareWithError` in order to add HTTP headers
// provided by the rule's data.
func NewAddSecurityHeadersCallbacks(rule Context, nextProlog, nextEpilog sqhook.Callback) (prolog, epilog sqhook.Callback, err error) {
	var headers http.Header
	if cfg := rule.Config(); cfg != nil {
		cfg, ok := rule.Config().([]interface{})
		if !ok {
			err = sqerrors.Errorf("unexpected callback data type: got `%T` instead of `[][]string`", cfg)
			return
		}
		headers = make(http.Header, len(cfg))
		for _, headersKV := range cfg {
			// TODO: move to a structured list of headers to avoid these dynamic type checking
			kv, ok := headersKV.([]string)
			if !ok {
				err = sqerrors.Errorf("unexpected number of values: header key and values are expected but got `%d` values instead", len(kv))
				return
			}
			if len(kv) != 2 {
				err = sqerrors.Errorf("unexpected number of values: header key and values are expected but got `%d` values instead", len(kv))
				return
			}
			headers.Set(kv[0], kv[1])
		}
	}
	if len(headers) == 0 {
		return nil, nil, sqerrors.New("there are no headers to add")
	}

	// Next callbacks to call
	actualNextProlog, ok := nextProlog.(AddSecurityHeadersPrologCallbackType)
	if nextProlog != nil && !ok {
		err = sqerrors.Errorf("unexpected next prolog type `%T` instead of `%T`", nextProlog, AddSecurityHeadersPrologCallbackType(nil))
		return
	}
	// No epilog in this callback, so simply check and pass the given one
	if _, ok := nextEpilog.(AddSecurityHeadersEpilogCallbackType); nextEpilog != nil && !ok {
		err = sqerrors.Errorf("unexpected next epilog type `%T` instead of `%T`", nextEpilog, AddSecurityHeadersEpilogCallbackType(nil))
		return
	}
	return newAddHeadersPrologCallback(headers, actualNextProlog), nextEpilog, nil
}

type AddSecurityHeadersPrologCallbackType = func(*sqhook.Context, *http.ResponseWriter) error
type AddSecurityHeadersEpilogCallbackType = func(*sqhook.Context)

// The prolog callback modifies the function arguments in order to replace the
// written status code and body.
func newAddHeadersPrologCallback(headers http.Header, next AddSecurityHeadersPrologCallbackType) AddSecurityHeadersPrologCallbackType {
	return func(ctx *sqhook.Context, w *http.ResponseWriter) error {
		responseHeaders := (*w).Header()
		for k, v := range headers {
			responseHeaders[k] = v
		}

		if next == nil {
			return nil
		}
		return next(ctx, w)
	}
}

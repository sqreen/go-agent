// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// NewAddSecurityHeadersCallbacks returns the native prolog and epilog callbacks
// to be hooked to `sqhttp.MiddlewareWithError` in order to add HTTP headers
// provided by the rule's data.
func NewAddSecurityHeadersCallbacks(rule Context, nextProlog sqhook.PrologCallback) (prolog interface{}, err error) {
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
		return nil, sqerrors.New("there are no headers to add")
	}

	// Next callbacks to call
	actualNextProlog, ok := nextProlog.(AddSecurityHeadersPrologCallbackType)
	if nextProlog != nil && !ok {
		err = sqerrors.Errorf("unexpected next prolog type `%T` instead of `%T`", nextProlog, AddSecurityHeadersPrologCallbackType(nil))
		return
	}
	return newAddHeadersPrologCallback(headers, actualNextProlog), nil
}

type AddSecurityHeadersEpilogCallbackType = func(*error)
type AddSecurityHeadersPrologCallbackType = func(*http.ResponseWriter) (AddSecurityHeadersEpilogCallbackType, error)

// The prolog callback modifies the function arguments in order to replace the
// written status code and body.
func newAddHeadersPrologCallback(headers http.Header, next AddSecurityHeadersPrologCallbackType) AddSecurityHeadersPrologCallbackType {
	return func(w *http.ResponseWriter) (AddSecurityHeadersEpilogCallbackType, error) {
		responseHeaders := (*w).Header()
		for k, v := range headers {
			responseHeaders[k] = v
		}
		if next == nil {
			return nil, nil
		}
		return next(w)
	}
}

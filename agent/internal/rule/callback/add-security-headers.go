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
func NewAddSecurityHeadersCallbacks(data []interface{}) (prolog, epilog sqhook.Callback, err error) {
	var headers = make(http.Header, len(data))
	for _, headersKV := range data {
		// TODO: move to a structured list of headers to avoid dynamic type checking
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
	if len(headers) == 0 {
		return nil, nil, sqerrors.New("there are no headers to add")
	}
	return newAddHeadersPrologCallback(headers), nil, nil
}

type AddSecurityHeadersPrologCallbackType = func(*sqhook.Context, *http.ResponseWriter) error

// The prolog callback modifies the function arguments in order to replace the
// written status code and body.
func newAddHeadersPrologCallback(headers http.Header) AddSecurityHeadersPrologCallbackType {
	return func(_ *sqhook.Context, w *http.ResponseWriter) error {
		responseHeaders := (*w).Header()
		for k, v := range headers {
			responseHeaders[k] = v
		}
		return nil
	}
}

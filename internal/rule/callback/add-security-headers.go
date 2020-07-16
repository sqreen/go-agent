// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"net/http"

	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/sqlib/sqassert"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

// NewAddSecurityHeadersCallback returns the native prolog and epilog callbacks
// to be attached to compatible HTTP protection middlewares such as
// `protection/http`. It adds HTTP headers provided by the rule's configuration.
func NewAddSecurityHeadersCallback(rule RuleFace, cfg NativeCallbackConfig) (sqhook.PrologCallback, error) {
	sqassert.NotNil(rule, cfg)
	var headers http.Header
	data, ok := cfg.Data().([]interface{})
	if !ok {
		return nil, sqerrors.Errorf("unexpected callback data type: got `%T` instead of `[][]string`", data)
	}
	headers = make(http.Header, len(data))
	for _, headersKV := range data {
		// TODO: move to a structured list of headers to avoid these dynamic type checking
		kv, ok := headersKV.([]string)
		if !ok {
			return nil, sqerrors.Errorf("unexpected number of values: header key and values are expected but got `%d` values instead", len(kv))
		}
		if len(kv) != 2 {
			return nil, sqerrors.Errorf("unexpected number of values: header key and values are expected but got `%d` values instead", len(kv))
		}
		headers.Set(kv[0], kv[1])
	}
	if len(headers) == 0 {
		return nil, sqerrors.New("unexpected empty list of headers to add")
	}

	return newAddHeadersPrologCallback(headers), nil
}

type AddSecurityHeadersPrologCallbackType = httpprotection.NonBlockingPrologCallbackType
type AddSecurityHeadersEpilogCallbackType = httpprotection.NonBlockingEpilogCallbackType

// The prolog callback modifies the function arguments in order to replace the
// written status code and body.
func newAddHeadersPrologCallback(headers http.Header) AddSecurityHeadersPrologCallbackType {
	return func(p **httpprotection.RequestContext) (httpprotection.NonBlockingEpilogCallbackType, error) {
		ctx := *p
		responseHeaders := ctx.ResponseWriter.Header()
		for k, v := range headers {
			responseHeaders[k] = v
		}
		return nil, nil
	}
}

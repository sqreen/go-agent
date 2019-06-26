// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package httphandler

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

var writeResponseHook *sqhook.Hook

func init() {
	writeResponseHook = sqhook.New(WriteResponse)
}

// WriteResponse writes an HTTP response according to the given arguments.
// The statusCode is the only mandatory argument. Headers and body can be nil.
func WriteResponse(w http.ResponseWriter, r *http.Request, headers http.Header, statusCode int, body []byte) {
	{
		type Prolog = func(*sqhook.Context, *http.ResponseWriter, **http.Request, *http.Header, *int, *[]uint8) error
		type Epilog = func(*sqhook.Context)
		ctx := sqhook.Context{}
		prolog, epilog := writeResponseHook.Callbacks()
		if epilog, ok := epilog.(Epilog); ok {
			defer epilog(&ctx)
		}
		if prolog, ok := prolog.(Prolog); ok {
			err := prolog(&ctx, &w, &r, &headers, &statusCode, &body)
			if err != nil {
				return
			}
		}
	}

	if len(headers) != 0 {
		responseHeaders := w.Header()
		for k, v := range headers {
			responseHeaders[k] = v
		}
	}
	w.WriteHeader(statusCode)
	if len(body) != 0 {
		_, _ = w.Write(body)
	}
}

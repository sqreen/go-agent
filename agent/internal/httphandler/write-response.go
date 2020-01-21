// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package httphandler

import (
	"net/http"
)

// WriteResponse writes an HTTP response according to the given arguments.
// The statusCode is the only mandatory argument. Headers and body can be nil.
//go:noinline
func WriteResponse(w http.ResponseWriter, r *http.Request, headers http.Header, statusCode int, body []byte) {
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

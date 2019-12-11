// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor

import (
	"net/http"
	"net/url"
)

// BindingAccessorContext is wrapper type of a request providing
// the binding accessor interface expected by rules:
//   - `#.Request` returns the request binding accessor context
type BindingAccessorContext struct {
	Request *HTTPRequestBindingAccessorContext
}

func MakeBindingAccessorContext(req *http.Request, clientIP string) BindingAccessorContext {
	return BindingAccessorContext{
		Request: NewHTTPRequestBindingAccessorContext(req, clientIP),
	}
}

// HTTPRequestBindingAccessorContext is wrapper type of a request providing
// the binding accessor interface expected by rules:
// - every http.Request field (eg. `.Method`, `.Header`, etc.) can be accessed.
// - `.ClientIP` returns the agent-computed client IP address.
// - `.FilteredParams` returns the URL query values and form values in a
//   *HTTPRequestFilteredParams structure.
type HTTPRequestBindingAccessorContext struct {
	*http.Request
	ClientIP string
	cache    httpRequestBindingAccessorCache
}

type httpRequestBindingAccessorCache struct {
	filteredParams *HTTPRequestFilteredParams
}

type HTTPRequestFilteredParams struct {
	// Form contains the Form field value of a http.Request after calling
	// `ParseForm()`, which includes request values such as URLquery values or
	// url-encoded form values. The multi-part form data is not included.
	Form url.Values
}

// NewHTTPRequestBindingAccessorContext wraps a request and its computed client
// ip address into a type providing the expected interface by rules.
func NewHTTPRequestBindingAccessorContext(req *http.Request, clientIP string) *HTTPRequestBindingAccessorContext {
	return &HTTPRequestBindingAccessorContext{
		Request:  req,
		ClientIP: clientIP,
	}
}

func (r *HTTPRequestBindingAccessorContext) FilteredParams() *HTTPRequestFilteredParams {
	if r.cache.filteredParams == nil {
		_ = r.Request.ParseForm()
		r.cache.filteredParams = &HTTPRequestFilteredParams{
			Form: r.Request.Form,
		}
	}
	return r.cache.filteredParams
}

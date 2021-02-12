// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package http

import (
	"net/textproto"

	"github.com/sqreen/go-agent/internal/protection/http/types"
)

// RequestBindingAccessorContext is wrapper type of a request providing
// the binding accessor interface expected by rules:
// - every embedded types.RequestReader method (eg. `.Method`, `.Header`,
//   etc.) can be accessed.
// - `.FilteredParams` returns the URL query and form values in a
//   FilteredRequestParams structure.
type RequestBindingAccessorContext struct {
	types.RequestBindingAccessorReader
}

func NewRequestBindingAccessorContext(r types.RequestBindingAccessorReader) *RequestBindingAccessorContext {
	return &RequestBindingAccessorContext{RequestBindingAccessorReader: r}
}

func (r *RequestBindingAccessorContext) FilteredParams() types.RequestParamMap {
	return r.Params()
}

func (r *RequestBindingAccessorContext) Header(h string) ([]string, error) {
	headers := r.Headers()
	if headers == nil {
		return nil, nil
	}
	h = textproto.CanonicalMIMEHeaderKey(h)
	v, exists := headers[h]
	if !exists {
		return nil, nil
	}
	return v, nil
}

func (r *RequestBindingAccessorContext) Body() RequestBodyBindingAccessorContext {
	return r.RequestBindingAccessorReader.Body()
}

type RequestBodyBindingAccessorContext []byte

func (b RequestBodyBindingAccessorContext) String() string { return string(b) }
func (b RequestBodyBindingAccessorContext) Bytes() []byte  { return b }

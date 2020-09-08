// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package http

import (
	"context"

	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

// RequestBindingAccessorContext is wrapper type of a request providing
// the binding accessor interface expected by rules:
// - every embedded types.RequestReader method (eg. `.Method`, `.Header`,
//   etc.) can be accessed.
// - `.FilteredParams` returns the URL query and form values in a
//   FilteredRequestParams structure.
type RequestBindingAccessorContext struct {
	types.RequestReader
}

type (
	// RequestParamsSet represents the set of parameters present in the HTTP request,
	// such as URL query parameters, form values, cookies, etc. Every source of
	// parameter is stored in its map key to avoid conflicts
	// (cf. methods `Params()` and `FilteredParams()`).
	// We are intentionally not using `url.Values` because of the JS VM constraint
	// discussed here https://github.com/dop251/goja/issues/134: `url.Values` has
	// methods and the key enumeration of such value results in the list of
	// methods. Casting the type solves the issue, but it is not possible to cast
	// nested type definitions like `RequestParamsSet` without copying the map and
	// casting its values. To avoid this, we decided to change the type we return
	// from binding accessors to types not having methods.
	RequestParamMap = map[string][]interface{}
)

func NewRequestBindingAccessorContext(r types.RequestReader) *RequestBindingAccessorContext {
	return &RequestBindingAccessorContext{RequestReader: r}
}

func (r *RequestBindingAccessorContext) FromContext(v interface{}) (*RequestBindingAccessorContext, error) {
	if r.RequestReader != nil {
		return r, nil
	}
	ctx, ok := v.(context.Context)
	if !ok {
		return nil, sqerrors.Errorf("unexpected argument type `%T`: type `context.Context` expected", v)
	}
	reqCtx := FromContext(ctx)
	if reqCtx == nil {
		return nil, sqerrors.Errorf("could not get the http protection context from the context value `%#+v`: did you pass the request context?", v)
	}
	r.RequestReader = reqCtx.RequestReader
	return r, nil
}

func (r *RequestBindingAccessorContext) FilteredParams() RequestParamMap {
	queryForm := r.QueryForm()
	postForm := r.PostForm()
	params := r.RequestReader.Params()

	res := make(types.RequestParamMap, 2+len(params))
	if len(postForm) > 0 {
		res.Add("PostForm", postForm)
	}
	if len(queryForm) > 0 {
		res.Add("QueryForm", queryForm)
	}
	for k, v := range params {
		res.Add(k, v)
	}
	return res
}

func (r *RequestBindingAccessorContext) Params() RequestParamMap {
	return r.FilteredParams()
}

func (r *RequestBindingAccessorContext) Header(h string) (*string, error) {
	return r.RequestReader.Header(h), nil
}

func (r *RequestBindingAccessorContext) Body() RequestBodyBindingAccessorContext {
	return r.RequestReader.Body()
}

type RequestBodyBindingAccessorContext []byte

func (b RequestBodyBindingAccessorContext) String() string { return string(b) }
func (b RequestBodyBindingAccessorContext) Bytes() []byte  { return b }

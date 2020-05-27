// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package http

import (
	"context"
	"net/url"

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
	// from binding accessors to types not having method helpers.
	RequestParamsSet = map[string]RequestParams
	RequestParams    = map[string][]string
)

// Static assert that `url.Values` can be assigned to `RequestParams`.
var _ RequestParams = url.Values(nil)

func NewRequestBindingAccessorContext(r types.RequestReader) *RequestBindingAccessorContext {
	return &RequestBindingAccessorContext{RequestReader: r}
}

func (c *RequestBindingAccessorContext) FromContext(v interface{}) (*RequestBindingAccessorContext, error) {
	if c.RequestReader != nil {
		return c, nil
	}
	ctx, ok := v.(context.Context)
	if !ok {
		return nil, sqerrors.Errorf("unexpected argument type `%T`: type `context.Context` expected", v)
	}
	reqCtx := FromContext(ctx)
	if reqCtx == nil {
		return nil, sqerrors.Errorf("could not get the http protection context from the context value `%#+v`: did you pass the request context?", v)
	}
	c.RequestReader = reqCtx.RequestReader
	return c, nil
}

func (r *RequestBindingAccessorContext) FilteredParams() RequestParamsSet {
	set := RequestParamsSet{}
	if form := r.RequestReader.Form(); len(form) > 0 {
		set["Form"] = form
	}
	if framework := r.RequestReader.FrameworkParams(); len(framework) > 0 {
		set["Framework"] = framework
	}
	return set
}

func (r *RequestBindingAccessorContext) Params() RequestParamsSet {
	params := r.FilteredParams()
	// TODO: cookies, etc.
	return params
}

func (r *RequestBindingAccessorContext) Header(h string) (*string, error) {
	return r.RequestReader.Header(h), nil
}

// Helper types for callbacks who must be designed for this protection so that
// they are the source of truth and so that the compiler catches type issues
// when compiling (versus when the callback is attached).
type (
	NonBlockingPrologCallbackType        = func(**RequestContext) (NonBlockingEpilogCallbackType, error)
	BlockingPrologCallbackType           = func(**RequestContext) (BlockingEpilogCallbackType, error)
	IdentifyUserPrologCallbackType       = func(**RequestContext, *map[string]string) (BlockingEpilogCallbackType, error)
	ResponseMonitoringPrologCallbackType = func(**RequestContext, *types.ResponseFace) (NonBlockingEpilogCallbackType, error)
	NonBlockingEpilogCallbackType        = func()
	BlockingEpilogCallbackType           = func(*error)
)

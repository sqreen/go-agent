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

type RequestParams map[string]url.Values

func (set RequestParams) Add(key string, params url.Values) {
	if len(params) > 0 {
		set[key] = params
	}
}

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
		return nil, sqerrors.Errorf("could not get the http protection context from the context values: did you pass the request context?", v)
	}
	c.RequestReader = reqCtx.RequestReader
	return c, nil
}

func (r *RequestBindingAccessorContext) FilteredParams() RequestParams {
	set := RequestParams{}
	set.Add("Form", r.RequestReader.Form())
	set.Add("Framework", r.RequestReader.FrameworkParams())
	return set
}

func (r *RequestBindingAccessorContext) Params() RequestParams {
	params := r.FilteredParams()
	// TODO: cookies, etc.
	return params
}

func (r *RequestBindingAccessorContext) Header(h string) (string, error) {
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

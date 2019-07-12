// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package rule

import (
	"github.com/sqreen/go-agent/agent/internal/rule/callback"
	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

// CallbackConstructorFunc is a function returning a callback function
// configured with the given data. The data types are known by the constructor
// that can type-assert them.
type CallbacksConstructorFunc func(data []interface{}) (prolog, epilog sqhook.Callback, err error)

// NewCallbacks returns the prolog and epilog callbacks of the given callback
// name. And error is returned if the callback name is unknown.
func NewCallbacks(name string, data []interface{}) (prolog, epilog sqhook.Callback, err error) {
	var callbacksCtor CallbacksConstructorFunc
	switch name {
	default:
		return nil, nil, sqerrors.Errorf("undefined callback name `%s`", name)
	case "WriteCustomErrorPage":
		callbacksCtor = callback.NewWriteCustomErrorPageCallbacks
	case "WriteHTTPRedirection":
		callbacksCtor = callback.NewWriteHTTPRedirectionCallbacks
	case "AddSecurityHeaders":
		callbacksCtor = callback.NewAddSecurityHeadersCallbacks
	}
	return callbacksCtor(data)
}

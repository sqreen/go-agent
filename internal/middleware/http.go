// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package middleware

import (
	"net/http"

	"github.com/sqreen/go-agent/agent/types"
	"github.com/sqreen/go-agent/sdk"
	"golang.org/x/xerrors"
)

// Handler is equivalent to http.Handler but returns an error when the request
// should no longer be handled.
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request) error
}

type ProtectedRequest struct {
	*http.Request
	eventRecord types.RequestRecord
}

type ProtectedResponseWriter struct {
	Status int
	http.ResponseWriter
}

// HTTPRequestRecordContextKey is a context key. It can be used in HTTP
// handlers with context.Context.Value() to access the HTTPRequestRecord that
// was associated with the request by the middleware. The associated value will
// be of type *HTTPRequestRecord.
var ContextKey = contextKey{"sqreen.rr"}

// ContextKey allows to insert context values avoiding string collisions. Cf.
// `context.WithValue()`.
type contextKey struct {
	// This string value must be used by middleware functions whose framework
	// expects context keys of type string, such as Gin. `sdk.FromContext()`
	// expect this behaviour to fallback to string keys when getting the value
	// from the pointer address returned null.
	String string
}

func isHandlerError(err error) bool {
	// If the returned error is not nil nor a security response, return it now.
	var secResponse sdk.SecurityResponseMatch
	if err != nil && !xerrors.As(err, &secResponse) {
		return err
	}
}

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"database/sql"
	"reflect"
	"strings"

	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	http_protection_types "github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqgo"
	"github.com/sqreen/go-agent/internal/sqlib/sqsql"
)

func NewReflectedCallbackBindingAccessorContext(capabilities []string, p ProtectionContext, args, res []reflect.Value, ruleValues interface{}) (*BindingAccessorContextType, error) {
	var ctx = &BindingAccessorContextType{}
	for _, cap := range capabilities {
		switch cap {
		case "rule":
			ctx.Rule = NewRuleBindingAccessorContext(ruleValues)
		case "sql":
			ctx.SQL = NewSQLBindingAccessorContext()
		case "func":
			ctx.Func = NewFunctionBindingAccessorContext(args, res)
		case "request":
			baCtx, err := NewRequestBindingAccessorContext(p)
			if err != nil {
				return nil, sqerrors.Wrapf(err, "could not create the request binding accessor context")
			}
			ctx.HTTPRequestBindingAccessorContext = baCtx
		case "lib":
			ctx.Lib = NewLibraryBindingAccessorContext()
		case "cache":
			ctx.BindingAccessorResultCache = MakeBindingAccessorResultCache()
		default:
			return nil, sqerrors.Errorf("unknown binding accessor capability `%s`", cap)
		}
	}
	return ctx, nil
}

type RuleBindingAccessorContextType struct {
	Data RuleDataBindingAccessorContextType
}

type RuleDataBindingAccessorContextType struct {
	Values interface{}
}

func NewRuleBindingAccessorContext(values interface{}) *RuleBindingAccessorContextType {
	return &RuleBindingAccessorContextType{
		Data: RuleDataBindingAccessorContextType{
			Values: values,
		},
	}
}

func NewFunctionBindingAccessorContext(args []reflect.Value, rets []reflect.Value) *FuncCallBindingAccessorContextType {
	c := &FuncCallBindingAccessorContextType{}
	if l := len(args); l > 0 {
		c.Args = make([]interface{}, l)
		for i, arg := range args {
			c.Args[i] = arg.Elem().Interface()
		}
	}

	if l := len(rets); l > 0 {
		c.Rets = make([]interface{}, l)
		for i, ret := range rets {
			c.Rets[i] = ret.Elem().Interface()
		}
	}

	return c
}

func NewSQLBindingAccessorContext() *SQLBindingAccessorContextType {
	return &SQLBindingAccessorContextType{}
}

func (*SQLBindingAccessorContextType) Dialect(db *sql.DB, dialects map[string]interface{}) (string, error) {
	drv := sqsql.Unwrap(db.Driver())
	if drv == nil {
		type errKey struct{}
		return "", sqerrors.WithKey(sqerrors.New("unexpected nil SQL driver"), errKey{})
	}

	// Get the actual unreferenced type so that we can get its package path.
	drvType := reflect.ValueOf(drv).Type()
loop:
	for {
		switch drvType.Kind() {
		case reflect.Ptr, reflect.Interface:
			drvType = drvType.Elem()
		default:
			break loop
		}
	}

	pkgPath := sqgo.Unvendor(drvType.PkgPath())
	if pkgPath == "" {
		return "", sqerrors.Errorf("could not get the package path of driver type `%T`", db.Driver())
	}

	for dialect, pkgList := range dialects {
		pkgPaths, ok := pkgList.([]interface{})
		if !ok {
			return "", sqerrors.Errorf("unexpected type `%T` while expecting `%T`", pkgList, pkgPaths)
		}

		for i := range pkgPaths {
			path, ok := pkgPaths[i].(string)
			if !ok {
				return "", sqerrors.Errorf("unexpected type `%T` while expecting `%T`", pkgPaths[i], path)
			}
			if strings.HasPrefix(pkgPath, path) {
				return dialect, nil
			}
		}
	}
	return "", sqerrors.Errorf("could not detect the sql dialect of package `%s`", pkgPath)
}

// BindingAccessorContextType is the context passed to binding accessor calls of
// security rules. Its fields are instantiated according to the rule
// capabilities and are nil by default. This mainly allows to avoid their
// creation cost when not needed.
type BindingAccessorContextType struct {
	Lib  *LibraryBindingAccessorContextType
	Func *FuncCallBindingAccessorContextType
	SQL  *SQLBindingAccessorContextType
	Rule *RuleBindingAccessorContextType
	*HTTPRequestBindingAccessorContext
	BindingAccessorResultCache
}

type WAFBindingAccessorContextType struct {
	HTTPRequestBindingAccessorContext
	BindingAccessorResultCache
}

func MakeWAFCallbackBindingAccessorContext(c CallbackContext) (WAFBindingAccessorContextType, error) {
	switch protCtx := c.ProtectionContext().(type) {
	case *http_protection.ProtectionContext:
		return makeHTTPWAFCallbackBindingAccessorContext(protCtx.RequestReader), nil
	default:
		return WAFBindingAccessorContextType{}, sqerrors.Errorf("unexpected protection context type `%T`", protCtx)
	}
}

func makeHTTPWAFCallbackBindingAccessorContext(request http_protection_types.RequestReader) WAFBindingAccessorContextType {
	return WAFBindingAccessorContextType{
		HTTPRequestBindingAccessorContext: MakeHTTPRequestBindingAccessorContext(request),
		BindingAccessorResultCache:        MakeBindingAccessorResultCache(),
	}
}

type FuncCallBindingAccessorContextType struct {
	Args []interface{}
	Rets []interface{}
}

type SQLBindingAccessorContextType struct{}

type HTTPRequestBindingAccessorContext struct {
	Request *http_protection.RequestBindingAccessorContext
}

// BindingAccessorResultCache is a simple result cache. There is no result
// invalidation here as this first iteration is about caching results per
// call site, meaning that a new cache should be used every time a new
// binding accessor context is created.
type BindingAccessorResultCache map[string]interface{}

func MakeBindingAccessorResultCache() BindingAccessorResultCache {
	cache := make(BindingAccessorResultCache)
	return cache
}

func (b BindingAccessorResultCache) Set(expr string, value interface{}) {
	if b == nil {
		return
	}
	b[expr] = value
}

func (b BindingAccessorResultCache) Get(expr string) (value interface{}, exists bool) {
	if b == nil {
		return nil, false
	}

	value, exists = b[expr]
	return
}

func NewRequestBindingAccessorContext(p ProtectionContext) (*HTTPRequestBindingAccessorContext, error) {
	switch actual := p.(type) {
	default:
		return nil, sqerrors.Errorf("unexpected request type `%T`", actual)

	case *http_protection.ProtectionContext:
		return NewHTTPRequestBindingAccessorContext(actual.RequestReader), nil
	}
}

func NewHTTPRequestBindingAccessorContext(req http_protection_types.RequestReader) *HTTPRequestBindingAccessorContext {
	ctx := MakeHTTPRequestBindingAccessorContext(req)
	return &ctx
}

func MakeHTTPRequestBindingAccessorContext(request http_protection_types.RequestReader) HTTPRequestBindingAccessorContext {
	return HTTPRequestBindingAccessorContext{
		Request: http_protection.NewRequestBindingAccessorContext(request),
	}
}

// Library of functions accessible to binding accessor expressions
type (
	LibraryBindingAccessorContextType struct {
		Array ArrayLibraryBindingAccessorContextType
	}

	ArrayLibraryBindingAccessorContextType struct{}
)

func NewLibraryBindingAccessorContext() *LibraryBindingAccessorContextType {
	return &LibraryBindingAccessorContextType{}
}

// Prepend inserts the value into the first position of the slice.
func (ArrayLibraryBindingAccessorContextType) Prepend(slice, value interface{}) (interface{}, error) {
	// Create a new slice of the same type, having l+1 element capacity
	var (
		sv = reflect.ValueOf(slice)
		vv = reflect.ValueOf(value)

		newSlice     reflect.Value
		newSliceType reflect.Type
		newSliceLen  int
	)

	if slice == nil {
		// Create a new slice based on the value type
		newSliceLen = 1
		newSliceType = reflect.SliceOf(vv.Type())
	} else {
		// Create a new slice based on the slice type
		newSliceLen = sv.Len() + 1
		newSliceType = sv.Type()
	}

	newSlice = reflect.MakeSlice(newSliceType, newSliceLen, newSliceLen)

	// Insert the value first in the new slice
	newSlice.Index(0).Set(vv)

	// Add the slice values next
	for i := 1; i < newSliceLen; i++ {
		newSlice.Index(i).Set(sv.Index(i - 1))
	}

	return newSlice.Interface(), nil
}

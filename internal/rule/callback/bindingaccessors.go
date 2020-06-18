// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"database/sql"
	"reflect"
	"strings"

	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqgo"
)

func NewReflectedCallbackBindingAccessorContext(capabilities []string, args, res []reflect.Value, req types.RequestReader, values interface{}) (*BindingAccessorContextType, error) {
	var c = &BindingAccessorContextType{}
	for _, cap := range capabilities {
		switch cap {
		case "rule":
			c.Rule = NewRuleBindingAccessorContext(values)
		case "sql":
			c.SQL = NewSQLBindingAccessorContext()
		case "func":
			c.Func = NewFunctionBindingAccessorContext(args, res)
		case "request":
			c.RequestBindingAccessorContext = NewRequestBindingAccessorContext(req)
		case "lib":
			c.Lib = NewLibraryBindingAccessorContext()
		case "cache":
			c.BindingAccessorResultCache = MakeBindingAccessorResultCache()
		default:
			return nil, sqerrors.Errorf("unknown binding accessor capability `%s`", cap)
		}
	}
	return c, nil
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
			c.Args[i] = arg.Interface()
		}
	}

	if l := len(rets); l > 0 {
		c.Rets = make([]interface{}, l)
		for i, ret := range rets {
			c.Rets[i] = ret.Interface()
		}
	}

	return c
}

func NewSQLBindingAccessorContext() *SQLBindingAccessorContextType {
	return &SQLBindingAccessorContextType{}
}

func (*SQLBindingAccessorContextType) Dialect(dbPtr **sql.DB, dialects map[string]interface{}) (string, error) {
	db := *dbPtr
	drvType := reflect.ValueOf(db.Driver()).Type()
	pkgPath := sqgo.Unvendor(drvType.PkgPath())

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
	*RequestBindingAccessorContext
	BindingAccessorResultCache
}

type WAFBindingAccessorContextType struct {
	RequestBindingAccessorContext
	BindingAccessorResultCache
}

func MakeWAFCallbackBindingAccessorContext(request types.RequestReader) WAFBindingAccessorContextType {
	return WAFBindingAccessorContextType{
		RequestBindingAccessorContext: MakeRequestBindingAccessorContext(request),
		BindingAccessorResultCache:    MakeBindingAccessorResultCache(),
	}
}

type FuncCallBindingAccessorContextType struct {
	Args []interface{}
	Rets []interface{}
}

type SQLBindingAccessorContextType struct{}

type RequestBindingAccessorContext struct {
	Request *httpprotection.RequestBindingAccessorContext
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

func NewRequestBindingAccessorContext(request types.RequestReader) *RequestBindingAccessorContext {
	ctx := MakeRequestBindingAccessorContext(request)
	return &ctx
}

func MakeRequestBindingAccessorContext(request types.RequestReader) RequestBindingAccessorContext {
	return RequestBindingAccessorContext{
		Request: httpprotection.NewRequestBindingAccessorContext(request),
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

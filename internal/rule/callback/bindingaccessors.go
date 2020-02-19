// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"database/sql"
	"reflect"
	"regexp"

	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

func NewCallbackBindingAccessorContext(capabilities []string, args, res []reflect.Value, req types.RequestReader) (*BindingAccessorContextType, error) {
	var (
		c   = &BindingAccessorContextType{}
		err error
	)
	for _, cap := range capabilities {
		switch cap {
		case "sql":
			c.SQL, err = NewSQLBindingAccessorContext()
			if err != nil {
				return nil, err
			}
		case "func":
			c.Func = NewFunctionBindingAccessorContext(args, res)
		case "request":
			c.RequestBindingAccessorContext = NewRequestCallbackBindingAccessorContext(req)
		default:
			return nil, sqerrors.Errorf("unknown binding accessor capability `%s`", cap)
		}
	}
	return c, nil
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

func NewSQLBindingAccessorContext() (*SQLBindingAccessorContextType, error) {
	return &SQLBindingAccessorContextType{}, nil
}

// TODO: make dynamic via the rule config
var dialects = map[string]*regexp.Regexp{
	"mysql":      regexp.MustCompile(`(?i)(my.*sql)`),
	"postgresql": regexp.MustCompile(`(?i)(pg)|(pq)|(post)`),
	"sqlite":     regexp.MustCompile(`(?i)(lite)`),
	"oracle":     regexp.MustCompile(`(?i)(ora)`),
}

func (*SQLBindingAccessorContextType) Dialect(v interface{}) (string, error) {
	db, ok := v.(*sql.DB)
	if !ok {
		return "", sqerrors.Errorf("unexpected type `%T` while expecting `*sql.DB`", v)
	}
	drvType := reflect.ValueOf(db.Driver()).Elem().Type()
	pkgPath := drvType.PkgPath()
	for dialect, re := range dialects {
		if re.MatchString(pkgPath) {
			return dialect, nil
		}
	}
	return "", sqerrors.Errorf("could not detect the sql dialect of package `%s`", pkgPath)
}

type BindingAccessorContextType struct {
	Func *FuncCallBindingAccessorContextType
	SQL  *SQLBindingAccessorContextType
	*RequestBindingAccessorContext
}

type FuncCallBindingAccessorContextType struct {
	Args []interface{}
	Rets []interface{}
}

type SQLBindingAccessorContextType struct{}

type RequestBindingAccessorContext struct {
	Request *httpprotection.RequestBindingAccessorContext
}

func NewRequestCallbackBindingAccessorContext(request types.RequestReader) *RequestBindingAccessorContext {
	ctx := &RequestBindingAccessorContext{}
	ctx.Request = httpprotection.NewRequestBindingAccessorContext(request)
	return ctx
}

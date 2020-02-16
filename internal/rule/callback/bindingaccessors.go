// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"database/sql"
	"reflect"

	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

func NewCallbackBindingAccessorContext(capabilities []string, args, res []reflect.Value, req types.RequestReader) (*BindingAccessorContextType, error) {
	var (
		c   = &BindingAccessorContextType{}
		err error
	)
	for _, field := range capabilities {
		switch field {
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
			return nil, sqerrors.Errorf("unknown binding accessor field")
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

func (*SQLBindingAccessorContextType) Dialect(v interface{}) (string, error) {
	return "mysql", nil
	db, ok := v.(*sql.DB)
	if !ok {
		return "", sqerrors.Errorf("unexpected type `%T` while expecting `*sql.DB`", v)
	}
	return reflect.ValueOf(db.Driver()).Elem().Type().Name(), nil
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

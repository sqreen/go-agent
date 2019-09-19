// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor

import (
	"reflect"

	"github.com/sqreen/go-agent/agent/sqlib/sqerrors"
)

func execIndexAccess(v interface{}, index interface{}) interface{} {
	lvalue := reflect.ValueOf(v)
	switch lvalue.Kind() {
	case reflect.Map:
		value := lvalue.MapIndex(reflect.ValueOf(index))
		var zero reflect.Value
		if value == zero {
			return nil
		}
		return value.Interface()
	case reflect.Slice:
		return lvalue.Index(index.(int)).Interface()
	default:
		panic(sqerrors.Errorf("cannot index value `%v` of type `%T` with index `%v` of type `%T`", v, v, index, index))
	}
}

func execFieldAccess(value interface{}, field string) interface{} {
	root := reflect.ValueOf(value)
	v := root
loop:
	for {
		switch v.Kind() {
		case reflect.Interface:
			fallthrough
		case reflect.Ptr:
			v = v.Elem()
		default:
			break loop
		}
	}

	// Try to access a field first
	zero := reflect.Value{}
	if f := v.FieldByName(field); f != zero {
		return f.Interface()
	}

	// Otherwise a method on the unreferenced value
	m := v.MethodByName(field)
	if m == zero && v != root {
		// Otherwise on the root value (in case of a pointer receiver)
		m = root.MethodByName(field)
	}

	if m == zero {
		panic(sqerrors.Errorf("no field nor method `%s` found in value of type `%T`", field, value))
	}

	// Call the the method which is expected to take no argument and to return a
	// single value. This line can panic on purpose as this is the primary way
	// of the `reflect` package for error management. Panics are caught by the
	// root binding accessor function and returned as an error.
	return m.Call(nil)[0].Interface()
}

func execFlatKeys(v interface{}) interface{} {
	return flatKeys(reflect.ValueOf(v))
}

func execFlatValues(v interface{}) interface{} {
	return flatValues(reflect.ValueOf(v))
}

func flatValues(v reflect.Value) []interface{} {
	var values []interface{}
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			values = append(values, flatValues(v.MapIndex(k))...)
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			values = append(values, flatValues(v.Field(i))...)
		}

	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			values = append(values, flatValues(v.Index(i))...)
		}

	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		return flatValues(v.Elem())

	default:
		values = []interface{}{v.Interface()}
	}

	return values
}

func flatKeys(v reflect.Value) []interface{} {
	var values []interface{}
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			values = append(values, k.Interface())
		}

	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			values = append(values, t.Field(i).Name)
			values = append(values, flatKeys(v.Field(i))...)
		}

	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			values = append(values, flatKeys(v.Index(i))...)
		}

	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		return flatKeys(v.Elem())

	default:
		values = []interface{}(nil)
	}

	return values
}

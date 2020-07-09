// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package bindingaccessor

import (
	"reflect"
	"unicode"

	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
)

func execIndexAccess(v interface{}, index interface{}) (interface{}, error) {
	lvalue := reflect.ValueOf(v)
doExecIndexAccess:
	switch lvalue.Kind() {
	case reflect.Func:
		// In order to be backward compatible with some binding accessor
		// expressions that were struct fields and became interface methods such
		// as `#.Request.Header['header']` that formerly was the http header map
		// and is now an interface method... To remove as soon as we deprecate
		// versions below v1.
		return execCall(v, index)
	case reflect.Map:
		value := lvalue.MapIndex(reflect.ValueOf(index))
		var zero reflect.Value
		if value == zero {
			return nil, nil
		}
		return value.Interface(), nil
	case reflect.Slice:
		return lvalue.Index(index.(int)).Interface(), nil
	case reflect.Ptr, reflect.Interface:
		lvalue = lvalue.Elem()
		goto doExecIndexAccess
	default:
		return nil, sqerrors.Errorf("cannot index value `%[1]v` of type `%[1]T` with index `%[2]v` of type `%[2]T`", v, index)
	}
}

func execFieldAccess(value interface{}, field string) (interface{}, error) {
	v := reflect.ValueOf(value)
	for {
		switch v.Kind() {
		case reflect.Interface, reflect.Ptr:
			if value, ok, err := tryExecMethodAccess(v, field); ok {
				return value, nil
			} else if err != nil {
				return nil, err
			}
			v = v.Elem()
			continue

		case reflect.Struct:
			// Try to access a field first
			if f := v.FieldByName(field); f.IsValid() {
				return f.Interface(), nil
			}
			fallthrough

		default:
			if value, ok, err := tryExecMethodAccess(v, field); ok {
				return value, nil
			} else if err != nil {
				return nil, err
			}
			return nil, sqerrors.Errorf("no field nor method `%s` found in value of type `%T`", field, value)
		}
	}
}

func tryExecMethodAccess(v reflect.Value, method string) (result interface{}, ok bool, err error) {
	m := v.MethodByName(method)

	if !m.IsValid() {
		return nil, false, nil
	}

	mt := m.Type()
	if mt.NumIn() != 0 {
		// Return the method interface value
		return m.Interface(), true, nil
	}

	res, err := execCall(m.Interface())
	return res, true, err
}

func execCall(fn interface{}, args ...interface{}) (interface{}, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if nbResults := fnType.NumOut(); nbResults != 1 && nbResults != 2 {
		return nil, sqerrors.Errorf("unexpected number of function results of function `%s`", fnType)
	} else if nbResults == 2 && !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, sqerrors.Errorf("unexpected second function results type of function `%s`: expected `error`", fnType)
	}

	argValues := make([]reflect.Value, len(args))
	for i, a := range args {
		if a != nil {
			argValues[i] = reflect.ValueOf(a)
		} else {
			// Don't use nil to avoid panics when calling the function with reflect.
			// Use instead the expected zero value for that argument type.
			argValues[i] = reflect.Zero(fnType.In(i))
		}
	}
	results := fnValue.Call(argValues)

	// return nil, err in case of an error
	if len(results) == 2 {
		if r1 := results[1]; !r1.IsNil() {
			return nil, r1.Interface().(error)
		}
	}

	// return the value otherwise
	var value interface{}
	switch r0 := results[0]; r0.Kind() {
	case reflect.Interface, reflect.Ptr:
		if !r0.IsNil() {
			value = r0.Interface()
		}
	default:
		value = r0.Interface()
	}
	return value, nil
}

func execFlatKeys(ctx Context, v interface{}, maxDepth, maxElements int) interface{} {
	if v == nil {
		return nil
	}
	return flatKeys(reflect.ValueOf(v), maxDepth, &maxElements)
}

func execFlatValues(ctx Context, v interface{}, maxDepth, maxElements int) interface{} {
	if v == nil {
		return nil
	}
	return flatValues(reflect.ValueOf(v), maxDepth, &maxElements)
}

func flatValues(v reflect.Value, depth int, elements *int) (values []interface{}) {
	switch v.Kind() {
	case reflect.Map:
		if depth == 0 {
			// do not traverse this value
			break
		}
		// Pre-allocate entries for at least one value per map entry
		values = make([]interface{}, 0, v.Len())
		for iter := v.MapRange(); iter.Next(); {
			values = append(values, flatValues(iter.Value(), depth-1, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Struct:
		if depth == 0 {
			// do not traverse this value
			break
		}
		t := v.Type()
		// Pre-allocate entries for at least one value per map entry
		l := v.NumField()
		values = make([]interface{}, 0, l)
		for i := 0; i < l; i++ {
			f := t.Field(i)
			if !unicode.IsUpper(rune(f.Name[0])) {
				// ignore private fields as their values cannot be traversed
				// (value.Field() panics).
				continue
			}
			values = append(values, flatValues(v.Field(i), depth-1, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Array, reflect.Slice:
		if depth == 0 {
			// do not traverse this value
			break
		}
		// Pre-allocate entries for at least one value per map entry
		l := v.Len()
		values = make([]interface{}, 0, l)
		for i := 0; i < l; i++ {
			values = append(values, flatValues(v.Index(i), depth-1, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Ptr:
		if v.IsNil() {
			return []interface{}{v.Interface()}
		}
		fallthrough
	case reflect.Interface:
		// do not count this step as a deeper level (no depth -= 1)
		return flatValues(v.Elem(), depth, elements)

	default:
		if !v.IsValid() || !v.CanInterface() {
			break
		}
		*elements -= 1
		values = []interface{}{v.Interface()}
	}

	if len(values) == 0 {
		return nil
	}
	return values
}

func flatKeys(v reflect.Value, depth int, elements *int) []interface{} {
	if depth == 0 || *elements == 0 {
		return nil
	}
	depth -= 1

	var values []interface{}
	switch v.Kind() {
	case reflect.Map:
		// Pre-allocate entries for at least one value per map entry
		l := v.Len()
		values = make([]interface{}, 0, l)
		for iter := v.MapRange(); iter.Next(); {
			k := iter.Key()
			values = append(values, k.Interface())
			*elements -= 1
			if *elements == 0 {
				break
			}
			values = append(values, flatKeys(iter.Value(), depth, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Struct:
		t := v.Type()
		// Pre-allocate entries for at least one value per map entry
		l := v.NumField()
		values = make([]interface{}, 0, l)
		for i := 0; i < l; i++ {
			f := t.Field(i)
			if !unicode.IsUpper(rune(f.Name[0])) {
				// ignore private fields as their values cannot be traversed
				// (value.Field() panics).
				continue
			}
			values = append(values, t.Field(i).Name)
			*elements -= 1
			values = append(values, flatKeys(v.Field(i), depth, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			values = append(values, flatKeys(v.Index(i), depth, elements)...)
			if *elements == 0 {
				break
			}
		}

	case reflect.Ptr, reflect.Interface:
		// traverse the interface and don't count this iteration in the depth count
		return flatKeys(v.Elem(), depth+1, elements)

	default:
		return nil
	}

	if len(values) == 0 {
		return nil
	}
	return values
}

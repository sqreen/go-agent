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
	switch lvalue.Kind() {
	case reflect.Func:
		// TODO: this is only for backward compatible with some binding accessor
		//   expressions that were struct fields and became interface methods such
		//   as `#.Request.Header['header']` that formerly was the http header map
		//   and is now an interface method... To remove as soon as we deprecate
		//   versions below v1.
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
	default:
		return nil, sqerrors.Errorf("cannot index value `%[1]v` of type `%[1]T` with index `%[2]v` of type `%[2]T`", v, index)
	}
}

func execFieldAccess(value interface{}, field string) (interface{}, error) {
	v := reflect.ValueOf(value)
	for {
		switch v.Kind() {
		case reflect.Interface:
			fallthrough
		case reflect.Ptr:
			if m := v.MethodByName(field); m.IsValid() {
				// Call the the method which is expected to take no argument and to return a
				// single value. This line can panic on purpose as this is the primary way
				// of the `reflect` package for error management. Panics are therefore caught
				// by the root binding accessor function and returned as an error.
				if m.Type().NumIn() == 0 {
					// TODO: validate and return instead an error if not possible
					return m.Call(nil)[0].Interface(), nil
				} else {
					return m.Interface(), nil
				}
			}
			v = v.Elem()
			continue

		case reflect.Struct:
			// Try to access a field first
			if f := v.FieldByName(field); f.IsValid() {
				return f.Interface(), nil
			}

			if m := v.MethodByName(field); m.IsValid() {
				// Call the the method which is expected to take no argument and to return a
				// single value. This line can panic on purpose as this is the primary way
				// of the `reflect` package for error management. Panics are therefore caught
				// by the root binding accessor function and returned as an error.
				if m.Type().NumIn() == 0 {
					// TODO: validate and return instead an error if not possible
					return m.Call(nil)[0].Interface(), nil
				} else {
					return m.Interface(), nil
				}
			}
			fallthrough

		default:
			return nil, sqerrors.Errorf("no field nor method `%s` found in value of type `%T`", field, value)
		}
	}
}

func execCall(fn interface{}, arg interface{}) (interface{}, error) {
	// TODO: func type validation
	results := reflect.ValueOf(fn).Call([]reflect.Value{reflect.ValueOf(arg)})

	if r1 := results[1]; !r1.IsNil() {
		return nil, r1.Interface().(error)
	}

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

	case reflect.Array:
		fallthrough
	case reflect.Slice:
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

	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
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

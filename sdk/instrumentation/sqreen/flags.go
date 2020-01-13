// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"reflect"
	"strings"
)

const structTagKey = "sqflag"

// parseFlags walks through the given arguments and sets the flagSet values
// present in the argument list. Unknown options, not present in the flagSet
// are accepted and skipped. The argument list is not modified.
func parseFlags(flagSet interface{}, args []string) {
	flagSetValueMap := makeFlagSetValueMap(flagSet)

	i := 0
	for i < len(args)-1 {
		opt, val, shift := parseOption(args[i], args[i+1])
		i += shift
		if f, exists := flagSetValueMap[opt]; exists {
			f.SetString(val)
		}
	}

	if i < len(args) {
		opt, val, _ := parseOption(args[i], "")
		if f, exists := flagSetValueMap[opt]; exists {
			f.SetString(val)
		}
	}
}

func makeFlagSetValueMap(flagSet interface{}) map[string]reflect.Value {
	v := reflect.ValueOf(flagSet).Elem()
	typ := v.Type()
	flagSetValueMap := make(map[string]reflect.Value, v.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag, ok := field.Tag.Lookup(structTagKey); ok {
			flagSetValueMap[tag] = v.Field(i)
		}
	}
	return flagSetValueMap
}

// parseOption parses the given current argument and following one according to
// the go flags syntax.
func parseOption(arg, nextArg string) (option, value string, shift int) {
	if arg[0] != '-' {
		// Not an option, return empty values and shift by one.
		shift = 1
		return
	}

	// Split the argument by its first `=` character if any, and check the
	// syntax being used.
	kv := strings.SplitN(arg, "=", 2)
	option = kv[0]
	if len(kv) == 2 {
		// `-opt=val` syntax
		value = kv[1]
		shift = 1
	} else if nextArg == "" || len(nextArg) > 1 && nextArg[0] != '-' {
		// `-opt val` syntax
		value = nextArg
		shift = 2
	} else {
		// `-opt` syntax (no value)
		shift = 1
	}

	return
}

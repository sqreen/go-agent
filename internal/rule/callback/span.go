// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package callback

import (
	"reflect"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
	sdk_types "github.com/sqreen/go-agent/sdk/types"
)

func MakeBlockingEpilog(errIndex int, err error) sqhook.ReflectedEpilogCallback {
	return func(results []reflect.Value) {
		ApplyBlockingError(results, errIndex, err)
	}
}

func ApplyBlockingError(results []reflect.Value, errIndex int, err error) {
	abortErr, ok := err.(sdk_types.SqreenError)
	if !ok {
		abortErr = sdk_types.SqreenError{Err: err}
	}
	results[errIndex].Elem().Set(reflect.ValueOf(abortErr))
}

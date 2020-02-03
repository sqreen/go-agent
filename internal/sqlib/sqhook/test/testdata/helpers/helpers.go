// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package helpers

import (
	"fmt"
	"log"
	"reflect"
	"runtime"

	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

func ShouldNotBeInstrumented(symbol string) {
	hook, err := sqhook.Find(symbol)
	if err != nil {
		log.Fatalln(err)
	}
	if hook != nil {
		log.Fatalf("symbol `%s` should not be instrumented but found hook `%s`", hook, symbol)
	}
}

func MustFind(symbol string) *sqhook.Hook {
	hook, err := sqhook.Find(symbol)
	if err != nil {
		log.Fatalln(err)
	}
	if hook == nil {
		log.Fatalf("no hook found for symbol `%s`", symbol)
	}
	return hook
}

func MustAttach(symbol string, prolog interface{}) {
	hook := MustFind(symbol)
	err := hook.Attach(prolog)
	if err != nil {
		log.Fatalf("could not attach `%T` to hook `%v`", prolog, hook)
	}
}

func MustAttachTracer(symbol string, prologType interface{}) {
	MustAttach(symbol, makePrologEpilogTracer(symbol, prologType))
}

//sqreen:ignore
//go:noinline
func getFunctionName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		panic("couldn't get the caller PC")
	}
	fname := runtime.FuncForPC(pc).Name()
	if fname == "" {
		panic("couln't read the function name")
	}
	return fname
}

func TraceCall() func() {
	fname := getFunctionName(2)
	return traceCall(fname)

}

func traceCall(fname string) func() {
	fmt.Println("IN:", fname)
	return func() {
		fmt.Println("OUT:", fname)
	}
}

func traceProlog(symbol string, params []reflect.Value) {
	traceCallback("PROLOG", symbol, params)
}

func traceEpilog(symbol string, results []reflect.Value) {
	traceCallback("EPILOG", symbol, results)
}

func traceCallback(where, symbol string, args []reflect.Value) {
	argFaces := make([]interface{}, len(args))
	for i, arg := range args {
		if arg.Kind() != reflect.Interface && arg.Kind() != reflect.Ptr {
			log.Fatalf("argument `%d` of `%s` of symbol `%s` is not a pointer but a `%T`", i, where, symbol, arg.Interface())
		}
		v := arg.Elem()
		var show interface{}
		if v.Kind() == reflect.Ptr {
			// Avoid showing addresses and rather show the type
			show = v.Type()
		} else {
			show = v.Interface()
		}
		argFaces[i] = show
	}
	fmt.Printf("%s: %s %v\n", where, symbol, argFaces)
}

func makePrologEpilogTracer(symbol string, prologType interface{}) interface{} {
	t := reflect.TypeOf(prologType)
	epilogType := t.Out(0)
	epilogFunc := reflect.MakeFunc(epilogType, func(results []reflect.Value) []reflect.Value {
		traceEpilog(symbol, results)
		return []reflect.Value{}
	})

	prologFunc := reflect.MakeFunc(t, func(params []reflect.Value) []reflect.Value {
		traceProlog(symbol, params)
		return []reflect.Value{epilogFunc, reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())}
	})

	return prologFunc.Interface()
}

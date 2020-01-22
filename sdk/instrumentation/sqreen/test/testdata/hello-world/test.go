// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package main

import (
	"fmt"
	"log"

	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

func init() {
	hook := mustFind("main.main")
	mustAttach(hook, func() (func(), error) {
		fmt.Println("IN: main.main")
		return func() {
			fmt.Println("OUT: main.main")
		}, nil
	})
}

func mustFind(symbol string) *sqhook.Hook {
	hook, err := sqhook.Find("main.main")
	if err != nil {
		panic(err)
	}
	if hook == nil {
		log.Fatalf("no hook found for symbol `%s`", symbol)
	}
	return hook
}

func mustAttach(hook *sqhook.Hook, prolog interface{}) {
	err := hook.Attach(prolog)
	if err != nil {
		log.Fatalf("could not attach `%T` to hook `%v`", prolog, hook)
	}
}

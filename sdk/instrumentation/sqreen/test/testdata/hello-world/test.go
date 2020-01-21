// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package main

import (
	"fmt"

	"github.com/sqreen/go-agent/agent/sqlib/sqhook"
)

func init() {
	hook, err := sqhook.Find("main.main")
	if err != nil {
		panic(err)
	}
	err = hook.Attach(func() (func(), error) {
		fmt.Println("IN: main.main")
		return func() {
			fmt.Println("OUT: main.main")
		}, nil
	})
	if err != nil {
		panic(err)
	}
}

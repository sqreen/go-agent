// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"fmt"

	"github.com/sqreen/go-agent/agent/internal/sqlib/sqhook/test/testdata/helpers"
)

func main() {
	defer helpers.TraceCall()()
	fmt.Println("Hello, Go!")
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

//sqreen:ignore

package main

import (
	"github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool/testdata/helpers"
)

func init() {
	helpers.MustAttachTracer("main.main", func() (func(), error)(nil))
}

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package main

import (
	"github.com/sqreen/go-agent/sdk/instrumentation/sqreen/test/testdata/helpers"
)

func init() {
	helpers.MustAttachTracer("main.main", func() (func(), error)(nil))
	helpers.MustAttachTracer("main.f1", func() (func(), error)(nil))
	helpers.MustAttachTracer("main.f2", func(*string) (func(), error)(nil))
	helpers.MustAttachTracer("main.f3", func(*string) (func(), error)(nil))
	helpers.MustAttachTracer("main.f4", func(*string, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.f5", func(*string, *string) (func(), error)(nil))
	helpers.MustAttachTracer("main.f6", func(*string) (func(), error)(nil))
	helpers.MustAttachTracer("main.f7", func(*string, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.f8", func(*string, *string, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.f9", func(*string, *string, *string) (func(), error)(nil))
	helpers.MustAttachTracer("main.f10", func(*string, *string, *string, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.f11", func(*[]interface{}) (func(), error)(nil))
	helpers.MustAttachTracer("main.f12", func() (func(*string), error)(nil))
	helpers.MustAttachTracer("main.f13", func() (func(*string), error)(nil))
	helpers.MustAttachTracer("main.f14", func() (func(*string, *int), error)(nil))
	helpers.MustAttachTracer("main.f15", func() (func(*string, *string), error)(nil))
	helpers.MustAttachTracer("main.f16", func() (func(*string), error)(nil))
	helpers.MustAttachTracer("main.f17", func() (func(*string, *int), error)(nil))
	helpers.MustAttachTracer("main.f18", func() (func(*string, *string, *int), error)(nil))
	helpers.MustAttachTracer("main.f19", func() (func(*string, *string, *string), error)(nil))
	helpers.MustAttachTracer("main.f20", func() (func(*string, *string, *string, *int), error)(nil))
	helpers.MustAttachTracer("main.f21", func(*int) (func(*int), error)(nil))
	helpers.MustAttachTracer("main.f22", func() (func(*int), error)(nil))
	helpers.MustAttachTracer("main.f23", func(*string, *int) (func(*bool, *int, *rune), error)(nil))
	helpers.ShouldNotBeInstrumented("main.f24")
	helpers.ShouldNotBeInstrumented("main.f25")
	helpers.MustAttachTracer("main.(s).m0", func(interface{}) (func(), error)(nil))
	helpers.MustAttachTracer("main.(s).m1", func(interface{}, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.(*s).m2", func(interface{}) (func(), error)(nil))
	helpers.MustAttachTracer("main.(*s).m3", func(interface{}) (func(), error)(nil))
	helpers.MustAttachTracer("main.(s).m4", func(interface{}) (func(), error)(nil))
	helpers.MustAttachTracer("main.(s).m5", func(interface{}, *int) (func(), error)(nil))
	helpers.MustAttachTracer("main.(s).m6", func(interface{}, *int) (func(), error)(nil))
}

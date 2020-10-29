// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// +build sqassert

package sqassert

import "github.com/sqreen/go-agent/internal/sqlib/sqerrors"

func True(c bool) {
	if !c {
		doPanic(sqerrors.New("sqassert: unexpected false value"))
	}
}

func False(c bool) {
	if c {
		doPanic(sqerrors.New("sqassert: unexpected true value"))
	}
}

func NoError(err error) {
	if err != nil {
		doPanic(sqerrors.Wrap(err, "unexpected error"))
	}
}

func NotNil(v ...interface{}) {
	for _, v := range v {
		if v == nil {
			doPanic(sqerrors.New("sqassert: unexpected nil value"))
		}
	}
}

func doPanic(err error) {
	panic(err)
}

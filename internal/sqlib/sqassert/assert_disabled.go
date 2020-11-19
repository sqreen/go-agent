// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// +build !sqassert

package sqassert

func True(bool)             {}
func False(bool)            {}
func NoError(error)         {}
func NotNil(...interface{}) {}

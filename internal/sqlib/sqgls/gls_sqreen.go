// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgls

import _ "unsafe" // for go:linkname

// get and set are the two functions allowing to access the GLS. In order to
// avoid linker errors and to allow executing non instrumented programs, we
// rely here on function variable values which can be nil so that we can detect
// at run time if the program was instrumented. When nil, the program wasn't;
// and when not nil, the program was and the two _sqreen_gls_get and
// _sqreen_gls_set can be used.
// When not instrumented, get and set defaults to no-op stubs.
var (
	get = func() interface{} { return nil }
	set = func(interface{}) {}
)

//go:linkname _sqreen_gls_get _sqreen_gls_get
var _sqreen_gls_get func() interface{}

//go:linkname _sqreen_gls_set _sqreen_gls_set
var _sqreen_gls_set func(interface{})

// Check at Go init time that the two function variable values created by the
// instrumentation tool are present, and set the get/set variables to their
// values.
func init() {
	if _sqreen_gls_get != nil && _sqreen_gls_set != nil {
		set = _sqreen_gls_set
		get = _sqreen_gls_get
	}
}

// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgls

import _ "unsafe" // for go:linkname

//go:linkname _sqreen_gls_get _sqreen_gls_get
func _sqreen_gls_get() interface{}

//go:linkname _sqreen_gls_set _sqreen_gls_set
func _sqreen_gls_set(interface{})

func get() interface{} { return _sqreen_gls_get() }

func set(v interface{}) { _sqreen_gls_set(v) }

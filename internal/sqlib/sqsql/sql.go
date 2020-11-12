// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqsql

import "database/sql/driver"

// Unwrapper is an interface that may have been implemented by SQL driver
// wrappers and allowing to get the underlying wrapped driver.
// Cf. https://github.com/elastic/apm-agent-go/issues/848 and
//     https://github.com/golang/go/issues/42460 for more details
type Unwrapper interface {
	Unwrap() driver.Driver
}

// Unwrap returns the deepest wrapped driver.
func Unwrap(d driver.Driver) driver.Driver {
	u, ok := d.(Unwrapper)
	for ok {
		d = u.Unwrap()
		u, ok = d.(Unwrapper)
	}
	return d
}

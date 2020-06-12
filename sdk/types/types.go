// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import "fmt"

// SqreenError is the wrapper error type returned by every Sqreen protection.
// It allows to implement specific error management when Sqreen has blocked a
// function call by returning a non-nil error. For example, if SQL-injection
// is detected and blocked by Sqreen, it is possible to avoid retrying the SQL
// query when the error is a `SqreenError`. It is possible to test if an error
// is a SqreenError by using `errors.As`.
// This type also implements the `Unwrap()` so that more details can be
// obtained by using `errors.As()`.
type SqreenError struct {
	Err error
}

func (e SqreenError) Error() string { return fmt.Sprintf("sqreen: %s", e.Err) }
func (e SqreenError) Unwrap() error { return e.Err }

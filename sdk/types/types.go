// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package types

import "fmt"

// SqreenError is the wrapper error type returned by every Sqreen protection.
// It allows to implement specific error management logic when Sqreen has
// blocked a function call by returning a non-nil Go error.
// For example, if a SQL injection was detected and blocked by Sqreen, it is
// possible to avoid retrying the SQL query when the error is a `SqreenError`.
// To do so, use `errors.As`:
//     if errors.As(err, &types.SqreenError{}) {
//         // Sqreen error logic
//         log.Println("sqreen error detected")
//     } else if err != nil {
//         // Non-Sqreen error logic
//     }
type SqreenError struct {
	Err error
}

func (e SqreenError) Error() string { return fmt.Sprintf("sqreen: %s", e.Err) }
func (e SqreenError) Unwrap() error { return e.Err }

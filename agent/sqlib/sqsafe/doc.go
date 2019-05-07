// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// This package provides functions making sure panics are safely caught and do
// not break the running program. Since panics can stop the program execution
// if they are not recovered, we need ways to safely recover and handle them.
//
// Therefore, this package provides simple function call wrappers to call a
// function that may panic and its goroutine equivalent. That way, any function
// can be safely called and any panic will be returned as a regular function
// error.
package sqsafe

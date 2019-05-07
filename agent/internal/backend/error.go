// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package backend

import "fmt"

type StatusError struct {
	StatusCode int
}

func NewStatusError(code int) *StatusError { return &StatusError{StatusCode: code} }
func (e *StatusError) Error() string {
	return fmt.Sprintf("http status error %d", e.StatusCode)
}

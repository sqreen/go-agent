// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package backend

import "fmt"

type HTTPStatusError struct {
	StatusCode int
}

func NewStatusError(code int) HTTPStatusError { return HTTPStatusError{StatusCode: code} }
func (e HTTPStatusError) Error() string {
	return fmt.Sprintf("http status error %d", e.StatusCode)
}

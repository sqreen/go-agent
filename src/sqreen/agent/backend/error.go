package backend

import "fmt"

type StatusError struct {
	StatusCode int
}

func NewStatusError(code int) *StatusError { return &StatusError{StatusCode: code} }
func (e *StatusError) Error() string {
	return fmt.Sprintf("http status error")
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package internal

type Event interface{}

type ExceptionEvent struct {
	err         error
	rulespackID string
}

func NewExceptionEvent(err error, rulespackID string) *ExceptionEvent {
	return &ExceptionEvent{err: err, rulespackID: rulespackID}
}

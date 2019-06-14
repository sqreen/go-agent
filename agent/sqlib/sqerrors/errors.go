// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqerrors

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/xerrors"
)

type Causer interface {
	Cause() error
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}

type Timestamper interface {
	Timestamp() time.Time
}

type withTimestamp struct {
	error
	timestamp time.Time
}

// WithTimestamp annotates the given error `err` with a timestamp. The returned
// error value implements interface Timestamper.
func WithTimestamp(err error) error {
	return &withTimestamp{
		error:     err,
		timestamp: time.Now(),
	}
}

func (e *withTimestamp) Timestamp() time.Time {
	return e.timestamp
}

func (e *withTimestamp) Unwrap() error {
	return e.error
}

func (e *withTimestamp) Cause() error {
	return e.Unwrap()
}

func (e *withTimestamp) Format(f fmt.State, c rune) {
	if formatter, ok := e.error.(fmt.Formatter); ok {
		formatter.Format(f, c)
	} else {
		_, _ = fmt.Fprintf(f, "%v", e.error)
	}
}

// New returns a new error annotated with a timestamp, a message and a stack
// trace.
func New(message string) error {
	return WithTimestamp(errors.New(message))
}

// Wrap annotates the given error `err` with a timestamp, a message and a stack
// trace.
func Wrap(err error, message string) error {
	return WithTimestamp(errors.Wrap(err, message))
}

// StackTrace returns the earliest/deepest StackTrace attached to any of
// the errors in the chain of Causes. If the error does not implement
// Cause, the original error will be returned. If the error is nil,
// nil will be returned without further investigation.
func StackTrace(err error) errors.StackTrace {
	var topStackInfo errors.StackTrace
loop:
	for {
		stackErr, ok := err.(StackTracer)
		if ok {
			topStackInfo = stackErr.StackTrace()
		}
		switch actual := err.(type) {
		case Causer:
			err = actual.Cause()
		case xerrors.Wrapper:
			err = actual.Unwrap()
		default:
			break loop
		}
	}
	return topStackInfo
}

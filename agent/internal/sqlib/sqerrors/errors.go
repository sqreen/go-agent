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

type Informer interface {
	Info() interface{}
}

type withInfo struct {
	error
	info interface{}
}

// WithInfo annotates the given error `err` with extra information giving more
// extra context to the error. The returned error value implements interface
// Info.
func WithInfo(err error, info interface{}) error {
	return &withInfo{
		error: err,
		info:  info,
	}
}

func (e *withInfo) Info() interface{} {
	return e.info
}

func (e *withInfo) Unwrap() error {
	return e.error
}

func (e *withInfo) Cause() error {
	return e.Unwrap()
}

// New returns a new error annotated with a timestamp, a message and a stack
// trace.
func New(message string) error {
	return WithTimestamp(errors.New(message))
}

// Errorf returns a new errors whose message is formatted by `fmt.Sprintf`. The
// returned error is annotated with a timestamp, a message and a stack trace.
func Errorf(format string, args ...interface{}) error {
	return New(fmt.Sprintf(format, args...))
}

// Wrap annotates the given error `err` with a timestamp, a message and a stack
// trace.
func Wrap(err error, message string) error {
	return WithTimestamp(errors.Wrap(err, message))
}

// Wrapf annotates the given error `err` with a timestamp, a message and a stack
// trace. The message is formatted by `fmt.Sprintf`.
func Wrapf(err error, format string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, args...))
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

// Info returns the earliest/deepest information attached to any of the errors
// in the chain of Causes. If the error does not implement Cause, the original
// error will be returned. If the error is nil, nil will be returned without
// further investigation.
func Info(err error) interface{} {
	var info interface{}
loop:
	for {
		infoErr, ok := err.(Informer)
		if ok {
			info = infoErr.Info()
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
	return info
}

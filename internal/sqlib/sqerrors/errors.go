// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqerrors

import (
	"fmt"
	"strings"
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
	return withTimestamp{
		error:     err,
		timestamp: time.Now(),
	}
}

func (e withTimestamp) Timestamp() time.Time {
	return e.timestamp
}

func (e withTimestamp) Unwrap() error {
	return e.error
}

func (e withTimestamp) Cause() error {
	return e.Unwrap()
}

func (e withTimestamp) Format(f fmt.State, c rune) {
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
	return withInfo{
		error: err,
		info:  info,
	}
}

func (e withInfo) Info() interface{} {
	return e.info
}

func (e withInfo) Unwrap() error {
	return e.error
}

func (e withInfo) Cause() error {
	return e.Unwrap()
}

type KeyType interface{}

type Keyer interface {
	Key() KeyType
}

type withKey struct {
	error
	key KeyType
}

// WithKey associates the given key with the error. This key can be used for
// error indexing in advanced error management cases such as error sampling.
func WithKey(err error, key KeyType) error {
	return withKey{
		error: err,
		key:   key,
	}
}

func (e withKey) Key() KeyType {
	return e.key
}

func (e withKey) Unwrap() error {
	return e.error
}

func (e withKey) Cause() error {
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

// StackTrace returns the deepest StackTrace attached to any of
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

// Info returns the earliest information attached to any of the errors
// in the chain of Causes. If the error does not implement Cause, the original
// error will be returned. If the error is nil, nil will be returned without
// further investigation.
func Info(err error) interface{} {
	for {
		switch actual := err.(type) {
		case Informer:
			return actual.Info()
		case Causer:
			err = actual.Cause()
		case xerrors.Wrapper:
			err = actual.Unwrap()
		default:
			return nil
		}
	}
}

// Timestamp returns the error timestamp created with the function
// `WithTimestamp()` and the `ok` return value set to true. Otherwise, the
// default time's zero value is returned and `ok` is false.
// false and the
func Timestamp(err error) (t time.Time, ok bool) {
	for {
		switch actual := err.(type) {
		case Timestamper:
			return actual.Timestamp(), true
		case Causer:
			err = actual.Cause()
		case xerrors.Wrapper:
			err = actual.Unwrap()
		default:
			return time.Time{}, false
		}
	}
}

// Key returns the deepest key attached to the error if any.
// TODO: combine every key together instead as a coordinate
func Key(err error) (k KeyType, exists bool) {
	for {
		if keyer, ok := err.(Keyer); ok {
			k = keyer.Key()
			exists = true
		}

		switch actual := err.(type) {
		case Causer:
			err = actual.Cause()
		case xerrors.Wrapper:
			err = actual.Unwrap()
		default:
			return k, exists
		}
	}
}

type ErrorCollection []error

func (c ErrorCollection) Error() string {
	var s strings.Builder
	s.WriteString("multiple errors occurred:")
	for i, e := range c {
		fmt.Fprintf(&s, " (error %d) %s;", i+1, e.Error())
	}
	// Return the build string without the trailing `;`
	return s.String()[:s.Len()-1]
}

func (c *ErrorCollection) Add(e error) {
	*c = append(*c, e)
}

func (c ErrorCollection) ToError() error {
	if len(c) == 0 {
		return nil
	}
	return c
}

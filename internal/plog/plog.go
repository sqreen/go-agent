// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// Implementation of simple logging interfaces efficient in production
// environments, aiming at being as fast as possible when disabled. The trick
// consists in changing the underlying implementation pointer with a disabled
// logger which does nothing when called. The call when disabled costs the
// underlying interface call indirection, equivalent to 2 method calls.

package plog

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sqreen/go-agent/internal/sqlib/sqassert/sqsync"
	"github.com/sqreen/go-agent/internal/sqlib/sqerrors"
	"github.com/sqreen/go-agent/internal/sqlib/sqsafe"
	"github.com/sqreen/go-agent/internal/sqlib/sqtime"
)

// LogLevel represents the log level. Higher levels include lowers.
type LogLevel int

const (
	// Disabled value.
	Disabled LogLevel = iota
	// Error logs.
	Error
	// Info to Error logs.
	Info
	// Debug to Error logs.
	Debug
)

// String representations of log levels.
const (
	DisabledString = "disabled"
	ErrorString    = "error"
	InfoString     = "info"
	DebugString    = "debug"
)

// LogLevel type stringer.
func (l LogLevel) String() string {
	switch l {
	case Error:
		return ErrorString
	case Info:
		return InfoString
	case Debug:
		return DebugString
	}
	return DisabledString
}

// ParseLogLevel returns the logger level corresponding to the string
// representation `level`. The returned LogLevel is Disabled when none matches.
func ParseLogLevel(level string) LogLevel {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case DebugString:
		return Debug
	case InfoString:
		return Info
	case ErrorString:
		return Error
	default:
		return Disabled
	}
}

// Logger structure wrapping logger interfaces, one per level.
type Logger struct {
	DebugLevelLogger
}

type (
	DebugLevelLogger interface {
		DebugLogger
		InfoLevelLogger
	}

	InfoLevelLogger interface {
		InfoLogger
		ErrorLevelLogger
	}

	ErrorLevelLogger ErrorLogger

	ErrorLogger interface {
		Error(err error)
	}

	InfoLogger interface {
		Info(v ...interface{})
		Infof(format string, v ...interface{})
	}

	DebugLogger interface {
		Debug(v ...interface{})
		Debugf(format string, v ...interface{})
	}
)

// NewLogger returns a Logger instance wrapping one logger instance per level.
// They can thus be individually enabled or disabled.
func NewLogger(level LogLevel, out io.Writer, errChan chan error) *Logger {
	var levelLogger DebugLevelLogger
	switch level {
	case Debug:
		levelLogger = debugLevelLogger{
			infoLevelLogger: infoLevelLogger{
				errorLevelLogger: newErrorLevelLogger(out, errChan, true),
			},
		}
	case Info:
		levelLogger = infoLevelLogger{
			errorLevelLogger: newErrorLevelLogger(out, errChan, true),
		}
	case Error:
		levelLogger = newErrorLevelLogger(out, errChan, false)
	default:
		levelLogger = makeDisabledLogger(errChan)
	}

	return &Logger{
		DebugLevelLogger: levelLogger,
	}
}

func newErrorLevelLogger(out io.Writer, errChan chan error, debugLevel bool) *errorLevelLogger {
	return &errorLevelLogger{
		writer: &logWriter{
			start: time.Now(),
			out:   out,
		},
		errChan:        errChan,
		debugLevel:     debugLevel,
		disabledLogger: makeDisabledLogger(errChan),
	}
}

type (
	debugLevelLogger struct {
		infoLevelLogger
	}

	infoLevelLogger struct {
		*errorLevelLogger
	}

	errorLevelLogger struct {
		disabledLogger
		writer *logWriter
		// Channel of logged errors, no matter if it is disabled.
		errChan    <-chan error
		debugLevel bool
	}

	disabledLogger struct {
		errChan chan error
	}
)

func (l debugLevelLogger) Debug(v ...interface{}) {
	l.writer.write(Debug, fmt.Sprint(v...))
}

func (l debugLevelLogger) Debugf(format string, v ...interface{}) {
	l.writer.write(Debug, fmt.Sprintf(format, v...))
}

func (l infoLevelLogger) Info(v ...interface{}) {
	l.writer.write(Info, fmt.Sprint(v...))
}

func (l infoLevelLogger) Infof(format string, v ...interface{}) {
	l.writer.write(Info, fmt.Sprintf(format, v...))
}

func (l *errorLevelLogger) Error(err error) {
	// Call disabledLogger's Error() for its error channel
	l.disabledLogger.Error(err)

	// Most detailed error format, including stacktrace when available.
	var format string
	if l.debugLevel {
		format = "%+v"
	} else {
		format = "%v"
	}
	l.writer.write(Error, fmt.Sprintf(format, err))
}

func makeDisabledLogger(errChan chan error) disabledLogger {
	return disabledLogger{
		errChan: errChan,
	}
}

func (l disabledLogger) Error(err error) {
	select {
	case l.errChan <- err:
	default:
	}
}
func (disabledLogger) Info(...interface{})           {}
func (disabledLogger) Infof(string, ...interface{})  {}
func (disabledLogger) Debug(...interface{})          {}
func (disabledLogger) Debugf(string, ...interface{}) {}

// Time formatting layout with microsecond precision.
const TimestampLayout = "2006-01-02T15:04:05.999999"

type logWriter struct {
	start time.Time
	out   io.Writer
}

func (l *logWriter) write(level LogLevel, message string) {
	var str strings.Builder
	str.WriteString("sqreen/")
	str.WriteString(level.String())
	str.WriteString(" - ")
	now := l.start.Add(time.Since(l.start)).Format(TimestampLayout)
	str.WriteString(now)
	str.WriteString(" - ")
	str.WriteString(message)
	str.WriteString("\n")
	_, _ = io.WriteString(l.out, str.String())
}

type backoffLogger struct {
	DebugLevelLogger
	// Map of sqtime.BackoffCounter counters
	counters sqsync.UInt64Map
	common   sqtime.BackoffCounter
}

func WithBackoff(logger DebugLevelLogger) DebugLevelLogger {
	if actual, ok := logger.(*backoffLogger); ok {
		return actual
	}

	// Don't backoff when in debug level mode
	if _, isDebugLevel := logger.(*debugLevelLogger); isDebugLevel {
		return logger
	}

	return &backoffLogger{
		DebugLevelLogger: logger,
	}
}

func (l *backoffLogger) Error(err error) {
	// Wrap the following in a safe call to avoid sync.Map indexing panic (eg.
	// non-comparable error key)
	safeCallErr := sqsafe.Call(func() error {
		var counter *sqtime.BackoffCounter
		if k, exists := sqerrors.Key(err); exists {
			counter = (*sqtime.BackoffCounter)(l.counters.Get(k))
		} else {
			counter = &l.common
		}

		counter.Do(func(_ uint64) {
			// TODO: use count in sqerrors.WithInfo()?
			l.DebugLevelLogger.Error(err)
		})

		return nil
	})

	if safeCallErr != nil {
		// A panic occurred so we fall back to the common counter but also append
		// the returned error to it
		l.common.Do(func(_ uint64) {
			l.DebugLevelLogger.Error(sqerrors.ErrorCollection{safeCallErr, err})
		})
	}
}

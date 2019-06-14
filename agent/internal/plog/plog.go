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
	ErrorLogger
	InfoLogger
	DebugLogger
	// Channel of logged errors, no matter if it is disabled.
	errChan chan error
}

type ErrorLogger interface {
	Error(err error)
}

type InfoLogger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
}

type DebugLogger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
}

// NewLogger returns a Logger instance wrapping one logger instance per level.
// They can thus be individually enabled or disabled.
func NewLogger(level LogLevel, output io.Writer, errChanBufferLen int) *Logger {
	disabled := disabledLogger{}
	var errChan chan error
	if errChanBufferLen > 0 {
		errChan = make(chan error, errChanBufferLen)
	}
	logger := &Logger{
		ErrorLogger: disabled,
		InfoLogger:  disabled,
		DebugLogger: disabled,
		errChan:     errChan,
	}
	enabled := enabledLogger{Writer: output, level: level}
	switch level {
	case Debug:
		logger.DebugLogger = enabled
		fallthrough
	case Info:
		logger.InfoLogger = enabled
		fallthrough
	case Error:
		logger.ErrorLogger = enabled
		break
	}
	return logger
}

// ErrChan returns the error channel. When Error logs are produced, the logged
// error is sent into the channel.
func (l *Logger) ErrChan() <-chan error {
	return l.errChan
}

// Error logs the error and send it into the error channel. If the channel is
// full, the send operation is dropped but the logging is still produced.
func (l *Logger) Error(err error) {
	// Non-blocking send into the error channel
	select {
	case l.errChan <- err:
	default:
	}
	l.ErrorLogger.Error(err)
}

// Enabled logger instance.
type enabledLogger struct {
	io.Writer
	level LogLevel
}

func (l enabledLogger) Debug(v ...interface{}) {
	_, _ = l.Write(formatLog(Debug, time.Now(), fmt.Sprint(v...)))
}

func (l enabledLogger) Debugf(format string, v ...interface{}) {
	_, _ = l.Write(formatLog(Debug, time.Now(), fmt.Sprintf(format, v...)))
}

func (l enabledLogger) Info(v ...interface{}) {
	_, _ = l.Write(formatLog(Info, time.Now(), fmt.Sprint(v...)))
}

func (l enabledLogger) Infof(format string, v ...interface{}) {
	_, _ = l.Write(formatLog(Info, time.Now(), fmt.Sprintf(format, v...)))
}

func (l enabledLogger) Error(err error) {
	// Most detailed error format, including stacktrace when available.
	var format string
	if l.level == Debug {
		format = "%+v"
	} else {
		format = "%v"
	}
	_, _ = l.Write(formatLog(Error, time.Now(), fmt.Sprintf(format, err)))
}

// Time formatting layout with microsecond precision.
const TimestampLayout = "2006-01-02T15:04:05.999999"

func formatLog(level LogLevel, now time.Time, message string) []byte {
	return []byte(fmt.Sprintf("sqreen/%s - %s - %s\n", level.String(), now.Format(TimestampLayout), message))
}

type disabledLogger struct {}

func (disabledLogger) Error(error)                   {}
func (disabledLogger) Info(...interface{})           {}
func (disabledLogger) Infof(string, ...interface{})  {}
func (disabledLogger) Debug(...interface{})          {}
func (disabledLogger) Debugf(string, ...interface{}) {}

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

const (
	// Error logs.
	ErrorString = "error"
	// Info to Error logs.
	InfoString = "info"
	// Debug to Error logs.
	DebugString = "debug"
)

// LogLevel type stringer.
func (l LogLevel) String() string {
	switch l {
	case Error:
		return "error"
	case Info:
		return "info"
	case Debug:
		return "debug"
	}
	return ""
}

// Logger structure wrapping logger interfaces, one per level.
type Logger struct {
	ErrorLogger
	InfoLogger
	DebugLogger
}

type ErrorLogger interface {
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
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
func NewLogger(level LogLevel, output io.Writer) *Logger {
	disabled := disabledLogger{}
	logger := &Logger{
		ErrorLogger: disabled,
		InfoLogger:  disabled,
		DebugLogger: disabled,
	}
	enabled := enabledLogger{output}
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

// Enabled logger instance.
type enabledLogger struct {
	io.Writer
}

func (l enabledLogger) Debug(v ...interface{}) {
	_, _ = l.Write(formatLog(Debug, time.Now(), fmt.Sprint(v...)))
}

func (l enabledLogger) Debugf(format string, v ...interface{}) {
	_, _ = l.Write(formatLog(Error, time.Now(), fmt.Sprintf(format, v...)))
}

func (l enabledLogger) Info(v ...interface{}) {
	_, _ = l.Write(formatLog(Info, time.Now(), fmt.Sprint(v...)))
}

func (l enabledLogger) Infof(format string, v ...interface{}) {
	_, _ = l.Write(formatLog(Error, time.Now(), fmt.Sprintf(format, v...)))
}

func (l enabledLogger) Error(v ...interface{}) {
	_, _ = l.Write(formatLog(Error, time.Now(), fmt.Sprint(v...)))
}

func (l enabledLogger) Errorf(format string, v ...interface{}) {
	_, _ = l.Write(formatLog(Error, time.Now(), fmt.Sprintf(format, v...)))
}

// Time formatting layout with microsecond precision.
const TimestampLayout = "2006-01-02T15:04:05.999999"

func formatLog(level LogLevel, now time.Time, message string) []byte {
	return []byte(fmt.Sprintf("sqreen/%s - %s - %s\n", level.String(), now.Format(TimestampLayout), message))
}

type disabledLogger struct {
}

func (_ disabledLogger) Error(_ ...interface{}) {
}
func (_ disabledLogger) Errorf(_ string, _ ...interface{}) {
}
func (_ disabledLogger) Info(_ ...interface{}) {
}
func (_ disabledLogger) Infof(_ string, _ ...interface{}) {
}
func (_ disabledLogger) Debug(_ ...interface{}) {
}
func (_ disabledLogger) Debugf(_ string, _ ...interface{}) {
}

// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package plog

import "strings"

// LogLevel represents the log level. Higher levels imply lowers.
type LogLevel int

const (
	// Disabled value.
	Disabled LogLevel = iota
	// Panic logs only.
	Panic
	// Error and Panic logs.
	Error
	// Info to Panic logs.
	Info
	// Debug to Panic logs.
	Debug
)

// String representation of log levels.
// Capitalized for better displaying the log messages as it makes new logger
// lines clearer.
const (
	DisabledString = "DISABLED"
	PanicString    = "PANIC"
	ErrorString    = "ERROR"
	InfoString     = "INFO"
	DebugString    = "DEBUG"
)

// LogLevel type stringer.
func (l LogLevel) String() string {
	switch l {
	case Disabled:
		return DisabledString
	case Panic:
		return PanicString
	case Error:
		return ErrorString
	case Info:
		return InfoString
	case Debug:
		return DebugString
	default:
		return ""
	}
}

// ParseLogLevel returns the logger level corresponding to the string
// representation `level`. The returned LogLevel is Disabled when none matches.
func ParseLogLevel(level string) LogLevel {
	switch strings.TrimSpace(strings.ToUpper(level)) {
	case DebugString:
		return Debug
	case InfoString:
		return Info
	case ErrorString:
		return Error
	case PanicString:
		return Panic
	default:
		return Disabled
	}
}

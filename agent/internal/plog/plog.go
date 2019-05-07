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
	"log"
	"sync"
)

// LogLevel represents the log level. Higher levels include lowers.
type LogLevel int

const (
	// Disabled value.
	Disabled LogLevel = iota
	// Panic logs.
	Panic
	// Error and Panic logs.
	Error
	// Warn to Panic logs.
	Warn
	// Info to Panic logs.
	Info
	// Debug to Panic logs.
	Debug
)

const (
	// Disabled value.
	DisabledString = "disabled"
	// Panic logs.
	PanicString = "fatal"
	// Error and Panic logs.
	ErrorString = "error"
	// Warn to Panic logs.
	WarnString = "warn"
	// Info to Panic logs.
	InfoString = "info"
	// Debug to Panic logs.
	DebugString = "debug"
)

// LogLevel type stringer.
func (l LogLevel) String() string {
	switch l {
	case Panic:
		return "fatal"
	case Error:
		return "error"
	case Warn:
		return "warn"
	case Info:
		return "info"
	case Debug:
		return "debug"
	}
	return ""
}

// Logger structure wrapping logger interfaces, one per level.
type Logger struct {
	PanicLogger
	ErrorLogger
	WarnLogger
	InfoLogger
	DebugLogger

	output    io.Writer
	namespace string
	cache     struct {
		fatal, error, warn, info, debug logger
	}
}

type PanicLogger interface {
	Panic(err error, v ...interface{})
	Panicf(err error, format string, v ...interface{})
	OutputSetter
}

type ErrorLogger interface {
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	OutputSetter
}

type WarnLogger interface {
	Warn(v ...interface{})
	Warnf(format string, v ...interface{})
	OutputSetter
}

type InfoLogger interface {
	Info(v ...interface{})
	Infof(format string, v ...interface{})
	OutputSetter
}

type DebugLogger interface {
	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	OutputSetter
}

type OutputSetter interface {
	SetOutput(output io.Writer)
}

// NewLogger returns a Logger instance wrapping one logger instance per level.
// They can thus be individually enabled or disabled.
func NewLogger(namespace string, parent *Logger) *Logger {
	logger := &Logger{
		PanicLogger: disabledLogger{},
		ErrorLogger: disabledLogger{},
		WarnLogger:  disabledLogger{},
		InfoLogger:  disabledLogger{},
		DebugLogger: disabledLogger{},
		namespace:   namespace,
	}

	if parent == nil {
		namespace = "sqreen/" + namespace
	} else {
		namespace = parent.namespace + "/" + namespace
		logger.SetLevel(parent.getLevel())
		logger.SetOutput(parent.output)
	}

	lock.Lock()
	defer lock.Unlock()
	loggers[namespace] = logger
	return logger
}

// SetOutput sets the output of all loggers created so far.
func SetOutput(output io.Writer) {
	for _, l := range loggers {
		l.SetOutput(output)
	}
}

// SetLevelFromString sets the logging level of all loggers created so far.
func SetLevelFromString(level string) {
	lvl := Disabled
	switch level {
	case DebugString:
		lvl = Debug
		break
	case InfoString:
		lvl = Info
		break
	case WarnString:
		lvl = Warn
		break
	case ErrorString:
		lvl = Error
		break
	case PanicString:
		lvl = Panic
		break
	}
	SetLevel(lvl)
}

// SetLevel sets the logging level of all loggers created so far.
func SetLevel(level LogLevel) {
	for _, l := range loggers {
		l.SetLevel(level)
	}
}

// SetOutput sets the output of the logger. When `nil`, the logger is disabled
// and equivalent to `SetLevel(Disabled)`.
func (l *Logger) SetOutput(output io.Writer) {
	l.output = output
	if output == nil {
		l.SetLevel(Disabled)
		return
	}
	l.PanicLogger.SetOutput(output)
	l.ErrorLogger.SetOutput(output)
	l.WarnLogger.SetOutput(output)
	l.InfoLogger.SetOutput(output)
	l.DebugLogger.SetOutput(output)
}

// SetLevelFromString change the level of the logger to `level`, possibly
// disabling it when the "disabled" string is passed.
func (l *Logger) SetLevelFromString(level string) {
	lvl := Disabled
	switch level {
	case DebugString:
		lvl = Debug
		break
	case InfoString:
		lvl = Info
		break
	case WarnString:
		lvl = Warn
		break
	case ErrorString:
		lvl = Error
		break
	case PanicString:
		lvl = Panic
		break
	}
	l.SetLevel(lvl)
}

// SetLevel changes the level of the logger to `level`, possibly disabling it
// when `Disabled` is passed.
func (l *Logger) getLevel() LogLevel {
	if _, disabled := l.DebugLogger.(disabledLogger); !disabled {
		return Debug
	}

	if _, disabled := l.InfoLogger.(disabledLogger); !disabled {
		return Info
	}

	if _, disabled := l.WarnLogger.(disabledLogger); !disabled {
		return Warn
	}

	if _, disabled := l.ErrorLogger.(disabledLogger); !disabled {
		return Error
	}

	if _, disabled := l.PanicLogger.(disabledLogger); !disabled {
		return Panic
	}

	return Disabled
}

// SetLevel changes the level of the logger to `level`, possibly disabling it
// when `Disabled` is passed.
func (l *Logger) SetLevel(level LogLevel) {
	switch level {
	case Disabled:
		l.DebugLogger = disabledLogger{}
		l.InfoLogger = disabledLogger{}
		l.WarnLogger = disabledLogger{}
		l.ErrorLogger = disabledLogger{}
		l.PanicLogger = disabledLogger{}
		break
	case Debug:
		l.DebugLogger = l.getLogger(Debug)
		l.InfoLogger = l.getLogger(Info)
		l.WarnLogger = l.getLogger(Warn)
		l.ErrorLogger = l.getLogger(Error)
		l.PanicLogger = l.getLogger(Panic)
		break
	case Info:
		l.DebugLogger = disabledLogger{}
		l.InfoLogger = l.getLogger(Info)
		l.WarnLogger = l.getLogger(Warn)
		l.ErrorLogger = l.getLogger(Error)
		l.PanicLogger = l.getLogger(Panic)
		break
	case Warn:
		l.DebugLogger = disabledLogger{}
		l.InfoLogger = disabledLogger{}
		l.WarnLogger = l.getLogger(Warn)
		l.ErrorLogger = l.getLogger(Error)
		l.PanicLogger = l.getLogger(Panic)
		break
	case Error:
		l.DebugLogger = disabledLogger{}
		l.InfoLogger = disabledLogger{}
		l.WarnLogger = disabledLogger{}
		l.ErrorLogger = l.getLogger(Error)
		l.PanicLogger = l.getLogger(Panic)
		break
	case Panic:
		l.DebugLogger = disabledLogger{}
		l.InfoLogger = disabledLogger{}
		l.WarnLogger = disabledLogger{}
		l.ErrorLogger = disabledLogger{}
		l.PanicLogger = l.getLogger(Panic)
		break
	}
}

// *log.Logger format prefix: UTC time, date of the day and time with microseconds.
const flags = log.LUTC | log.Ldate | log.Lmicroseconds

// Map of loggers.
var loggers = make(map[string]*Logger)
var lock sync.Mutex

// Enabled logger instance.
type logger struct {
	logger *log.Logger
}

// Return the logger instance. Create a new one if first time.
func (l *Logger) getLogger(level LogLevel) logger {
	cache := l.getCachedLogger(level)
	if cache.logger == nil {
		*cache = newLogger(l.namespace, level, l.output)
	} else {
		cache.SetOutput(l.output)
	}
	return *cache
}

// Return a new logger.
func newLogger(namespace string, level LogLevel, output io.Writer) logger {
	return logger{log.New(output, namespace+":"+level.String()+": ", flags)}
}

// Return the pointer to the cache entry.
func (l *Logger) getCachedLogger(level LogLevel) *logger {
	switch level {
	case Debug:
		return &l.cache.debug
	case Info:
		return &l.cache.info
	case Warn:
		return &l.cache.warn
	case Error:
		return &l.cache.error
	case Panic:
		return &l.cache.fatal
	}
	return nil
}

func (l logger) Panic(err error, v ...interface{}) {
	l.logger.Output(3, fmt.Sprint(v...))
	panic(err)
}

func (l logger) Panicf(err error, format string, v ...interface{}) {
	l.logger.Output(3, fmt.Sprintf(format, v...))
	panic(err)
}

func (l logger) Debug(v ...interface{}) {
	l.logger.Output(3, fmt.Sprint(v...))
}

func (l logger) Debugf(format string, v ...interface{}) {
	l.logger.Output(3, fmt.Sprintf(format, v...))
}

func (l logger) Info(v ...interface{}) {
	l.logger.Output(3, fmt.Sprint(v...))
}

func (l logger) Infof(format string, v ...interface{}) {
	l.logger.Output(3, fmt.Sprintf(format, v...))
}

func (l logger) Warn(v ...interface{}) {
	l.logger.Output(3, fmt.Sprint(v...))
}

func (l logger) Warnf(format string, v ...interface{}) {
	l.logger.Output(3, fmt.Sprintf(format, v...))
}

func (l logger) Error(v ...interface{}) {
	l.logger.Output(3, fmt.Sprint(v...))
}

func (l logger) Errorf(format string, v ...interface{}) {
	l.logger.Output(3, fmt.Sprintf(format, v...))
}

func (l logger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}

type disabledLogger struct {
}

func (_ disabledLogger) Panic(_ error, _ ...interface{}) {
}
func (_ disabledLogger) Panicf(_ error, _ string, _ ...interface{}) {
}
func (_ disabledLogger) Error(_ ...interface{}) {
}
func (_ disabledLogger) Errorf(_ string, _ ...interface{}) {
}
func (_ disabledLogger) Warn(_ ...interface{}) {
}
func (_ disabledLogger) Warnf(_ string, _ ...interface{}) {
}
func (_ disabledLogger) Info(_ ...interface{}) {
}
func (_ disabledLogger) Infof(_ string, _ ...interface{}) {
}
func (_ disabledLogger) Debug(_ ...interface{}) {
}
func (_ disabledLogger) Debugf(_ string, _ ...interface{}) {
}
func (_ disabledLogger) SetOutput(_ io.Writer) {
}

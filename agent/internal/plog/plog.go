// Implementation of simple logging interfaces efficient in production
// environments, aiming at being as fast as possible when disabled. The trick
// consists in changing the underlying implementation pointer with a disabled
// logger which does nothing when called. The call when disabled costs the
// underlying interface call indirection, equivalent to 2 method calls.

package plog

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type PanicLogger interface {
	Panic(err error, v ...interface{})
	Panicf(err error, format string, v ...interface{})
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

// Logger is the logger structure wrapping the distinct levels that can be each
// disabled.
type Logger struct {
	panicImpl PanicLogger
	errorImpl ErrorLogger
	infoImpl  InfoLogger
	debugImpl DebugLogger
}

// NewLogger returns a Logger instance wrapping one logger instance per level.
// They can thus be individually enabled or disabled.
func NewLogger(level LogLevel, output io.Writer) *Logger {
	l := &Logger{}
	disabled := disabledLogger{}
	enabled := enabledLogger{writer: output}
	switch level {
	default:
		l.debugImpl = disabled
		l.infoImpl = disabled
		l.errorImpl = disabled
		l.panicImpl = disabled
	case Debug:
		l.debugImpl = enabled
		l.infoImpl = enabled
		l.errorImpl = enabled
		l.panicImpl = enabled
	case Info:
		l.debugImpl = disabled
		l.infoImpl = enabled
		l.errorImpl = enabled
		l.panicImpl = enabled
	case Error:
		l.debugImpl = disabled
		l.infoImpl = disabled
		l.errorImpl = enabled
		l.panicImpl = enabled
	case Panic:
		l.debugImpl = disabled
		l.infoImpl = disabled
		l.errorImpl = disabled
		l.panicImpl = enabled
	}
	return l
}

func (l *Logger) Panic(err error, v ...interface{}) {
	l.panicImpl.Panic(err, v...)
	panic(err)
}

func (l *Logger) Panicf(err error, format string, v ...interface{}) {
	l.panicImpl.Panicf(err, format, v...)
	panic(err)
}

func (l *Logger) Debug(v ...interface{}) {
	l.debugImpl.Debug(v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.debugImpl.Debugf(format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.infoImpl.Info(v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.infoImpl.Infof(format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.errorImpl.Error(v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.errorImpl.Errorf(format, v...)
}

type enabledLogger struct {
	writer io.Writer
}

func (l enabledLogger) Panic(err error, v ...interface{}) {
	l.write(Panic, v...)
}

func (l enabledLogger) Panicf(err error, format string, v ...interface{}) {
	l.writef(Panic, format, v...)
}

func (l enabledLogger) Debug(v ...interface{}) {
	l.write(Debug, v...)
}

func (l enabledLogger) Debugf(format string, v ...interface{}) {
	l.writef(Debug, format, v...)
}

func (l enabledLogger) Info(v ...interface{}) {
	l.write(Info, v...)
}

func (l enabledLogger) Infof(format string, v ...interface{}) {
	l.writef(Info, format, v...)
}

func (l enabledLogger) Error(v ...interface{}) {
	l.write(Error, v...)
}

func (l enabledLogger) Errorf(format string, v ...interface{}) {
	l.writef(Error, format, v...)
}

func (l enabledLogger) writef(level LogLevel, format string, v ...interface{}) {

}

func (l enabledLogger) write(level LogLevel, args ...interface{}) {
	time := FormatTime(time.Now())
	text := fmt.Sprint(args...)
	_, _ = fmt.Fprintf(l.writer, "SQREEN/%s - %s - %s\n", level.String(), time, text)
}

func FormatTime(t time.Time) (buf []byte) {
	year := t.Year()
	month := t.Month()
	day := t.Day()
	hour := t.Hour()
	min := t.Minute()
	sec := t.Second()
	micro := t.Nanosecond() / 1000
	// The expected capacity of the array is set to the sum of base 10 values.
	// If this is not enough, the slice capacity will be increased by append().
	buf = make([]byte, 0, 4+2+2+2+2+2+6+6)
	buf = strconv.AppendUint(buf, uint64(year), 10)
	buf = append(buf, '-')
	buf = strconv.AppendUint(buf, uint64(month), 10)
	buf = append(buf, '-')
	buf = strconv.AppendUint(buf, uint64(day), 10)
	buf = append(buf, 'T')
	buf = strconv.AppendUint(buf, uint64(hour), 10)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, uint64(min), 10)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, uint64(sec), 10)
	buf = append(buf, '.')
	buf = strconv.AppendUint(buf, uint64(micro), 10)
	return
}

type disabledLogger struct{}

func (disabledLogger) Panic(error, ...interface{}) {
}
func (disabledLogger) Panicf(error, string, ...interface{}) {
}
func (disabledLogger) Error(...interface{}) {
}
func (disabledLogger) Errorf(string, ...interface{}) {
}
func (disabledLogger) Info(...interface{}) {
}
func (disabledLogger) Infof(string, ...interface{}) {
}
func (disabledLogger) Debug(...interface{}) {
}
func (disabledLogger) Debugf(string, ...interface{}) {
}

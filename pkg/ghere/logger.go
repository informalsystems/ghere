package ghere

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

type LogLevel int8

const (
	Debug LogLevel = 1
	Info  LogLevel = 2
	Warn  LogLevel = 3
	Error LogLevel = 4
)

// Logger is a common interface for our logging infrastructure.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

type zeroLogLogger struct {
	stdoutLogger *zerolog.Logger
	stderrLogger *zerolog.Logger
}

var _ Logger = (*zeroLogLogger)(nil)

// NewZerologLogger creates a Zerolog-based logger. See
// https://github.com/rs/zerolog for details.
func NewZerologLogger(level LogLevel) Logger {
	outw := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	errw := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}
	stdoutLogger := zerolog.New(outw).Level(level.zeroLogLevel()).With().Timestamp().Logger()
	stderrLogger := zerolog.New(errw).Level(level.zeroLogLevel()).With().Timestamp().Logger()
	return &zeroLogLogger{
		stdoutLogger: &stdoutLogger,
		stderrLogger: &stderrLogger,
	}
}

func (l *zeroLogLogger) Debug(msg string, keysAndValues ...interface{}) {
	if e := l.stdoutLogger.Debug(); e.Enabled() {
		applyZeroLogArgs(e, msg, keysAndValues)
	}
}

func (l *zeroLogLogger) Info(msg string, keysAndValues ...interface{}) {
	if e := l.stdoutLogger.Info(); e.Enabled() {
		applyZeroLogArgs(e, msg, keysAndValues)
	}
}

func (l *zeroLogLogger) Warn(msg string, keysAndValues ...interface{}) {
	if e := l.stdoutLogger.Warn(); e.Enabled() {
		applyZeroLogArgs(e, msg, keysAndValues)
	}
}

func (l *zeroLogLogger) Error(msg string, keysAndValues ...interface{}) {
	if e := l.stderrLogger.Error(); e.Enabled() {
		applyZeroLogArgs(e, msg, keysAndValues)
	}
}

func (lvl LogLevel) zeroLogLevel() zerolog.Level {
	switch {
	case lvl <= Debug:
		return zerolog.DebugLevel
	case lvl == Info:
		return zerolog.InfoLevel
	case lvl == Warn:
		return zerolog.WarnLevel
	}
	return zerolog.ErrorLevel
}

func applyZeroLogArgs(e *zerolog.Event, msg string, keysAndValues []interface{}) {
	var key string
	for i, kv := range keysAndValues {
		if i%2 == 0 {
			key = kv.(string)
		} else {
			switch v := kv.(type) {
			case error:
				e.Err(v)
			default:
				e.Interface(key, kv)
			}
		}
	}
	e.Msg(msg)
}

//-----------------------------------------------------------------------------

type noopLogger struct{}

var _ Logger = (*noopLogger)(nil)

func NewNoopLogger() Logger {
	return &noopLogger{}
}

// Debug implements Logger
func (*noopLogger) Debug(msg string, keysAndValues ...interface{}) {}

// Error implements Logger
func (*noopLogger) Error(msg string, keysAndValues ...interface{}) {}

// Info implements Logger
func (*noopLogger) Info(msg string, keysAndValues ...interface{}) {}

// Warn implements Logger
func (*noopLogger) Warn(msg string, keysAndValues ...interface{}) {}

// Package logging provides a configurable logger based on zerolog with options for log level,
// colored output, custom parts order, and time format.
//
// LogLevel represents the severity of the log message.
// Accepted values for log level (case-insensitive):
//   - "debug"
//   - "info"
//   - "warn"
//   - "error"
//   - "critical"
//
// Invalid log level values will be silently ignored and the default level will be used.
//
// Options can be set using functional options:
//   - WithLogLevel(level string): sets the log level. Accepted values are listed above.
//   - WithColors(): enables colored log output.
//   - WithPartsOrder(p []string): sets the order of log parts (fields).
//   - WithTimeFormat(t string): sets the time format for log timestamps.
//
// NewLogger(opts ...func(*options)) Logger creates a new logger instance with the provided options.
// If an option is not set, a default value will be used.
package logging

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Logger = zerolog.Logger
type Event = zerolog.Event

type LogLevel uint8

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	CRITICAL
)

type options struct {
	logLevel    zerolog.Level
	coloredLogs bool
	partsOrder  []string
	timeFormat  string
}

func WithLogLevel(level string) func(*options) {
	return func(o *options) {
		l := strings.ToLower(level)
		if logLevel, ok := map[string]zerolog.Level{
			"debug":    zerolog.DebugLevel,
			"info":     zerolog.InfoLevel,
			"warn":     zerolog.WarnLevel,
			"error":    zerolog.ErrorLevel,
			"critical": zerolog.FatalLevel,
		}[l]; ok {
			o.logLevel = logLevel
		}
	}
}

func WithColors() func(*options) {
	return func(o *options) {
		o.coloredLogs = true
	}
}

func WithPartsOrder(p []string) func(*options) {
	return func(o *options) {
		o.partsOrder = p
	}
}

func WithTimeFormat(t string) func(*options) {
	return func(o *options) {
		o.timeFormat = t
	}
}

func NewLogger(opts ...func(*options)) Logger {
	o := &options{
		logLevel:    zerolog.InfoLevel,
		coloredLogs: false,
		partsOrder:  []string{},
		timeFormat:  time.RFC3339,
	}

	for _, opt := range opts {
		opt(o)
	}
	zerolog.TimeFieldFormat = o.timeFormat
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !o.coloredLogs, TimeFormat: o.timeFormat}).Level(o.logLevel).With().Caller().Timestamp().Logger()
}

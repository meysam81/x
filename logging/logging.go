// Package logging provides a configurable zerolog-based logger with
// functional options for log level, colored output, parts order, and time format.
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

// WithLogLevel returns an option that sets the log level.
// Accepted values (case-insensitive): "debug", "info", "warn", "error", "critical".
// Unrecognized values are silently ignored and the default level (info) is used.
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

func WithColorsEnabled(enabled bool) func(*options) {
	return func(o *options) {
		o.coloredLogs = enabled
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

// NewLogger creates a Logger with the given functional options, falling back to sensible defaults.
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

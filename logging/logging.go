package logging

import (
	"os"
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
	logLevel    LogLevel
	coloredLogs bool
	partsOrder  []string
	timeFormat  string
}

func WithLogLevel(level LogLevel) func(*options) {
	return func(o *options) {
		o.logLevel = level
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

func detectLogLevel(level LogLevel) zerolog.Level {
	return map[LogLevel]zerolog.Level{
		DEBUG:    zerolog.DebugLevel,
		INFO:     zerolog.InfoLevel,
		WARN:     zerolog.WarnLevel,
		ERROR:    zerolog.ErrorLevel,
		CRITICAL: zerolog.FatalLevel,
	}[level]
}

func NewLogger(opts ...func(*options)) Logger {
	o := &options{
		logLevel:    INFO,
		coloredLogs: false,
		partsOrder:  []string{},
		timeFormat:  time.RFC3339,
	}

	for _, opt := range opts {
		opt(o)
	}
	level := detectLogLevel(o.logLevel)

	zerolog.TimeFieldFormat = o.timeFormat
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !o.coloredLogs, TimeFormat: o.timeFormat}).Level(level).With().Caller().Timestamp().Logger()
}

package logging

import (
	"io"
	"log"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
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

func detectLogLevel(level LogLevel) zerolog.Level {
	return map[LogLevel]zerolog.Level{
		DEBUG:    zerolog.DebugLevel,
		INFO:     zerolog.InfoLevel,
		WARN:     zerolog.WarnLevel,
		ERROR:    zerolog.ErrorLevel,
		CRITICAL: zerolog.FatalLevel,
	}[level]
}

func NewLogger(opts ...func(*options)) *Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	o := &options{
		logLevel:    INFO,
		coloredLogs: false,
		partsOrder:  []string{},
	}

	for _, opt := range opts {
		opt(o)
	}
	level := detectLogLevel(o.logLevel)

	var writer io.Writer = zerolog.ConsoleWriter{Out: os.Stderr, NoColor: !o.coloredLogs, PartsOrder: o.partsOrder}
	writer = diode.NewWriter(writer, 1000, 0, func(missed int) {
		log.Printf("Dropped %d messages", missed)
	})

	l := zerolog.New(writer).With().Caller().Timestamp().Logger().Level(level)
	return &l
}

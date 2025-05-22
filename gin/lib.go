package gin

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/meysam81/x/logging"
)

type Gin = gin.Engine

type options struct {
	keepDefaultWriter      bool
	keepDefaultErrorWriter bool
	errorHandler           *gin.HandlerFunc
	zerologDisabled        bool
	ginLoggerEnabled       bool
}

func WithKeepDefaultWriter() func(*options) {
	return func(o *options) {
		o.keepDefaultWriter = true
	}
}

func WithKeepDefaultErrorWriter() func(*options) {
	return func(o *options) {
		o.keepDefaultErrorWriter = true
	}
}

func WithCustomErrorHandler(h *gin.HandlerFunc) func(*options) {
	return func(o *options) {
		o.errorHandler = h
	}
}

func WithZeroLogDisabled() func(*options) {
	return func(o *options) {
		o.zerologDisabled = true
	}
}

func WithGinLoggerEnabled() func(*options) {
	return func(o *options) {
		o.ginLoggerEnabled = true
	}
}

func NewGin(logger *logging.Logger, opts ...func(*options)) *Gin {
	g := gin.New()

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if !o.keepDefaultWriter {
		gin.DefaultWriter = io.Discard
	}

	if !o.keepDefaultErrorWriter {
		gin.DefaultErrorWriter = io.Discard
	}

	if !o.zerologDisabled && logger != nil {
		g.Use(zerologMiddleware(logger))
	}

	if o.ginLoggerEnabled {
		g.Use(gin.Logger())
	}

	if o.errorHandler != nil {
		g.Use(*o.errorHandler)
	} else {
		g.Use(gin.Recovery())
	}

	return g
}

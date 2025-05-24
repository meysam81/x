package gin

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/meysam81/x/logging"
)

type Gin = gin.Engine

type options struct {
	keepDefaultWriter          bool
	keepDefaultErrorWriter     bool
	errorHandler               *gin.HandlerFunc
	ginLoggerEnabled           bool
	disableNullifyTrustedProxy bool
	logger                     *logging.Logger
	disableSetReleaseMode      bool
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

func WithGinLoggerEnabled() func(*options) {
	return func(o *options) {
		o.ginLoggerEnabled = true
	}
}

func WithZerologLogger(l *logging.Logger) func(*options) {
	return func(o *options) {
		o.logger = l
	}
}

func WithDisableSetModeRelease() func(*options) {
	return func(o *options) {
		o.disableSetReleaseMode = true
	}
}

func NewGin(opts ...func(*options)) *Gin {
	g := gin.New()

	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	if !o.disableNullifyTrustedProxy {
		g.SetTrustedProxies(nil)
	}

	gin.SetMode(gin.ReleaseMode)
	if !o.disableSetReleaseMode {
	}

	if !o.keepDefaultWriter {
		gin.DefaultWriter = io.Discard
	}

	if !o.keepDefaultErrorWriter {
		gin.DefaultErrorWriter = io.Discard
	}

	if o.logger != nil {
		g.Use(zerologMiddleware(o.logger))
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

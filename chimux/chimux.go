// Package chimux provides an opinionated chi router factory with built-in
// middleware for recovery, real IP, structured logging, Prometheus metrics,
// and health checks.
package chimux

import (
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/meysam81/x/logging"
)

type Option = func(*options)

type options struct {
	disableRecoveryMiddleware bool
	disableCleanPath          bool
	disableRealIP             bool

	enableLoggingMiddleware bool
	logger                  *logging.Logger
	headerLogMode           headerLogMode
	extraLogHeaders         map[string]struct{}

	enableMetrics        bool
	enableHealthz        bool
	enableHealthzLogging bool
	healthzEndpoint      string
	enableMetricsLogging bool
	metricsEndpoint      string
}

func WithDisableRecoveryMiddleware() Option {
	return func(o *options) {
		o.disableRecoveryMiddleware = true
	}
}

func WithDisableCleanPath() Option {
	return func(o *options) {
		o.disableCleanPath = true
	}
}

func WithDisableRealIP() Option {
	return func(o *options) {
		o.disableRealIP = true
	}
}

func WithLoggingMiddleware() Option {
	return func(o *options) {
		o.enableLoggingMiddleware = true
	}
}

func WithLogger(l *logging.Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

func WithDisableLogHeaders() Option {
	return func(o *options) {
		o.headerLogMode = headerLogNone
	}
}

func WithMetrics() Option {
	return func(o *options) {
		o.enableMetrics = true
	}
}

func WithHealthz() Option {
	return func(o *options) {
		o.enableHealthz = true
	}
}

func WithHealthEndpoint(uri string) Option {
	return func(o *options) {
		o.healthzEndpoint = uri
	}
}

func WithMetricsEndpoint(uri string) Option {
	return func(o *options) {
		o.metricsEndpoint = uri
	}
}

func WithLogHealthRequests() Option {
	return func(o *options) {
		o.enableHealthzLogging = true
	}
}

// WithLogAllHeaders configures the logger to include every request header.
// Sensitive headers are still masked.
func WithLogAllHeaders() Option {
	return func(o *options) {
		o.headerLogMode = headerLogAll
	}
}

// WithLogHeaders adds extra headers to the default logging set.
// Header names are case-insensitive.
func WithLogHeaders(headers ...string) Option {
	return func(o *options) {
		if o.extraLogHeaders == nil {
			o.extraLogHeaders = make(map[string]struct{})
		}
		for _, h := range headers {
			o.extraLogHeaders[strings.ToLower(h)] = struct{}{}
		}
	}
}

// NewChi creates a chi.Mux with opinionated defaults: CleanPath, RealIP, and
// Recoverer middleware are enabled out of the box. Use option functions to add
// structured logging, Prometheus metrics, health checks, or disable defaults.
func NewChi(opts ...Option) *chi.Mux {
	o := &options{
		disableRecoveryMiddleware: false,
		enableLoggingMiddleware:   false,
		headerLogMode:             headerLogDefault,
		enableMetrics:             false,
		healthzEndpoint:           "/healthz",
		metricsEndpoint:           "/metrics",
	}

	for _, opt := range opts {
		opt(o)
	}

	r := chi.NewRouter()

	if !o.disableCleanPath {
		r.Use(middleware.CleanPath)
	}

	if !o.disableRealIP {
		r.Use(middleware.RealIP)
	}

	if o.enableLoggingMiddleware {
		if o.logger == nil {
			l := logging.NewLogger()
			o.logger = &l
		}

		r.Use(loggingMiddleware(o))
	}

	if !o.disableRecoveryMiddleware {
		r.Use(middleware.Recoverer)
	}

	if o.enableMetrics {
		m := newMetrics()
		r.Use(m.middleware)
		r.Get(o.metricsEndpoint, promhttp.Handler().ServeHTTP)
	}

	if o.enableHealthz {
		r.Get(o.healthzEndpoint, healthCheck)
	}

	return r
}

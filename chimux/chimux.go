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

type Options struct {
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

func WithDisableRecoveryMiddleware() func(*Options) {
	return func(o *Options) {
		o.disableRecoveryMiddleware = true
	}
}

func WithDisableCleanPath() func(*Options) {
	return func(o *Options) {
		o.disableCleanPath = true
	}
}

func WithDisableRealIP() func(*Options) {
	return func(o *Options) {
		o.disableRealIP = true
	}
}

func WithLoggingMiddleware() func(*Options) {
	return func(o *Options) {
		o.enableLoggingMiddleware = true
	}
}

func WithLogger(l *logging.Logger) func(*Options) {
	return func(o *Options) {
		o.logger = l
	}
}

func WithDisableLogHeaders() func(*Options) {
	return func(o *Options) {
		o.headerLogMode = headerLogNone
	}
}

func WithMetrics() func(*Options) {
	return func(o *Options) {
		o.enableMetrics = true
	}
}

func WithHealthz() func(*Options) {
	return func(o *Options) {
		o.enableHealthz = true
	}
}

func WithHealthEndpoint(uri string) func(*Options) {
	return func(o *Options) {
		o.healthzEndpoint = uri
	}
}

func WithMetricsEndpoint(uri string) func(*Options) {
	return func(o *Options) {
		o.metricsEndpoint = uri
	}
}

func WithLogHealthRequests() func(*Options) {
	return func(o *Options) {
		o.enableHealthzLogging = true
	}
}

// WithLogAllHeaders configures the logger to include every request header.
// Sensitive headers are still masked.
func WithLogAllHeaders() func(*Options) {
	return func(o *Options) {
		o.headerLogMode = headerLogAll
	}
}

// WithLogHeaders adds extra headers to the default logging set.
// Header names are case-insensitive.
func WithLogHeaders(headers ...string) func(*Options) {
	return func(o *Options) {
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
func NewChi(opts ...func(*Options)) *chi.Mux {
	o := &Options{
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

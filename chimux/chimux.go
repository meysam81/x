package chimux

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/meysam81/x/logging"
)

type options struct {
	disableRecoveryMiddleware bool
	disableCleanPath          bool
	disableRealIP             bool
	enableLoggingMiddleware   bool
	logger                    *logging.Logger
	disableLogHeaders         bool
	enableMetrics             bool
	enableHealthz             bool

	enableHealthzLogging bool
	healthzEndpoint      string

	enableMetricsLogging bool
	metricsEndpoint      string
}

func WithDisableRecoveryMiddleware() func(*options) {
	return func(o *options) {
		o.disableRecoveryMiddleware = true
	}
}

func WithDisableCleanPath() func(*options) {
	return func(o *options) {
		o.disableCleanPath = true
	}
}

func WithDisableRealIP() func(*options) {
	return func(o *options) {
		o.disableRealIP = true
	}
}

func WithLoggingMiddleware() func(*options) {
	return func(o *options) {
		o.enableLoggingMiddleware = true
	}
}

func WithLogger(l *logging.Logger) func(*options) {
	return func(o *options) {
		o.logger = l
	}
}

func WithDisableLogHeaders() func(*options) {
	return func(o *options) {
		o.disableLogHeaders = true
	}
}

func WithMetrics() func(*options) {
	return func(o *options) {
		o.enableMetrics = true
	}
}

func WithHealthz() func(*options) {
	return func(o *options) {
		o.enableHealthz = true
	}
}

func WithHealthEndpoint(uri string) func(*options) {
	return func(o *options) {
		o.healthzEndpoint = uri
	}
}

func WithMetricsEndpoint(uri string) func(*options) {
	return func(o *options) {
		o.metricsEndpoint = uri
	}
}

func WithLogHealthRequests() func(*options) {
	return func(o *options) {
		o.enableHealthzLogging = true
	}
}

func NewChi(opts ...func(*options)) *chi.Mux {
	o := &options{
		disableRecoveryMiddleware: false,
		enableLoggingMiddleware:   false,
		disableLogHeaders:         false,
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

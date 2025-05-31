package chimux

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/meysam81/x/logging"
)

type options struct {
	disableRecoveryMiddleware bool
	disableLoggingMiddleware  bool
	logger                    *logging.Logger
	disableLogHeaders         bool
	disableMetrics            bool
	disableHealthz            bool
	healthzEndpoint           string
}

func WithDisableRecoveryMiddlware() func(*options) {
	return func(o *options) {
		o.disableRecoveryMiddleware = true
	}
}

func WithDisableLoggingMiddleware() func(*options) {
	return func(o *options) {
		o.disableLoggingMiddleware = true
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

func WithoutMetrics() func(*options) {
	return func(o *options) {
		o.disableMetrics = true
	}
}

func WithoutHealthEndpoint() func(*options) {
	return func(o *options) {
		o.disableHealthz = true
	}
}

func WithHealthEndpoint(pattern string) func(*options) {
	return func(o *options) {
		o.healthzEndpoint = pattern
	}
}

func NewChi(opts ...func(*options)) *chi.Mux {
	o := &options{
		disableRecoveryMiddleware: false,
		disableLoggingMiddleware:  false,
		disableLogHeaders:         false,
		disableMetrics:            false,
		healthzEndpoint:           "/healthz",
	}

	for _, opt := range opts {
		opt(o)
	}

	r := chi.NewRouter()

	if o.logger == nil && !o.disableLoggingMiddleware {
		o.logger = logging.NewLogger()
	}

	if !o.disableLoggingMiddleware {
		r.Use(loggingMiddleware(o.logger, !o.disableLogHeaders))
	}

	if !o.disableRecoveryMiddleware {
		r.Use(middleware.Recoverer)
	}

	if !o.disableMetrics {
		m := newMetrics()
		r.Use(m.middleware)
		r.Get("/metrics", promhttp.Handler().ServeHTTP)

	}

	if !o.disableHealthz {
		r.Get(o.healthzEndpoint, healthCheck)
	}

	return r
}

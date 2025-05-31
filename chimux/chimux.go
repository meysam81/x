package chimux

import (
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/meysam81/x/logging"
)

type options struct {
	disableRecoveryMiddleware bool
	disableLoggingMiddleware  bool
	logger                    *logging.Logger
	disableLogHeaders         bool
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

func NewChi(opts ...func(*options)) *chi.Mux {
	o := &options{
		disableRecoveryMiddleware: false,
		disableLoggingMiddleware:  false,
		disableLogHeaders:         false,
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

	return r
}

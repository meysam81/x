package chimux

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// routePattern returns the chi route pattern that served r, so the path
// label is bounded by the router's route table instead of by whatever the
// internet sends. Requests no route matched (404s, scanner probes) collapse
// into "unmatched". Falls back to the raw path only when chi did not run.
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return r.URL.Path
	}
	if p := rctx.RoutePattern(); p != "" {
		return p
	}
	return "unmatched"
}

// metricsSkipPaths returns the paths this router's RED middleware should
// not record: the metrics endpoint itself, plus healthz/readyz when
// enabled. Mirrors logRequest.shouldSkip's access-log exclusions, so a 2s
// liveness/readiness probe period does not show up as synthetic traffic in
// http_requests_total.
func metricsSkipPaths(o *options) map[string]struct{} {
	skip := map[string]struct{}{o.metricsEndpoint: {}}
	if o.enableHealthz {
		skip[o.healthzEndpoint] = struct{}{}
	}
	if o.enableReadyz {
		skip[o.readyzEndpoint] = struct{}{}
	}
	return skip
}

type metrics struct {
	// Traffic: Rate of requests
	httpRequestsTotal prometheus.CounterVec

	// Latency: Time taken to serve requests
	httpRequestDuration prometheus.HistogramVec

	// Errors: Rate of requests that fail
	httpResponseStatus prometheus.CounterVec

	// Saturation: Resource utilization
	httpRequestsInFlight prometheus.Gauge
}

var (
	sharedMetricsOnce sync.Once
	sharedMetricsInst *metrics
)

// sharedMetrics returns the process-wide RED metrics collectors, creating
// them on first use. promauto registers on Prometheus's global default
// registry (kept deliberately, so the default go_/process_ collectors still
// appear on /metrics), and registering the same collector name twice
// panics -- so every NewChi(WithMetrics()) router in this process shares
// this one instance instead of each constructing its own.
func sharedMetrics() *metrics {
	sharedMetricsOnce.Do(func() {
		sharedMetricsInst = newMetrics()
	})
	return sharedMetricsInst
}

func newMetrics() *metrics {
	return &metrics{
		httpRequestsTotal: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		httpRequestDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status_code"},
		),

		httpResponseStatus: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_response_status_total",
				Help: "Total number of HTTP responses by status code",
			},
			[]string{"status_code", "status_class"},
		),

		httpRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
	}
}

func (m *metrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	statusStr := strconv.Itoa(statusCode)
	statusClass := getStatusClass(statusCode)

	// Traffic: Increment request counter
	m.httpRequestsTotal.WithLabelValues(method, path, statusStr).Inc()

	// Latency: Record request duration
	m.httpRequestDuration.WithLabelValues(method, path, statusStr).Observe(duration.Seconds())

	// Errors: Record response status
	m.httpResponseStatus.WithLabelValues(statusStr, statusClass).Inc()
}

func (m *metrics) IncrementInFlight() {
	m.httpRequestsInFlight.Inc()
}

func (m *metrics) DecrementInFlight() {
	m.httpRequestsInFlight.Dec()
}

func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "1xx"
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// middleware returns chi middleware recording RED metrics for every request
// whose path is not in skip (e.g. the metrics/healthz/readyz endpoints of
// the router it is attached to -- see metricsSkipPaths). A skipped request
// still runs normally; it is just not counted, timed, or held against the
// in-flight gauge.
func (m *metrics) middleware(skip map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
			}

			defer func() {
				duration := time.Since(start)
				m.RecordRequest(r.Method, routePattern(r), wrapped.statusCode, duration)
			}()

			m.IncrementInFlight()
			defer m.DecrementInFlight()

			next.ServeHTTP(wrapped, r)
		})
	}
}

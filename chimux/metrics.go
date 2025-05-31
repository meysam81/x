package chimux

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	// Traffic: Rate of requests
	httpRequestsTotal prometheus.CounterVec

	// Latency: Time taken to serve requests
	httpRequestDuration prometheus.HistogramVec

	// Errors: Rate of requests that fail
	httpResponseStatus prometheus.CounterVec

	// Saturation: Resource utilization
	httpRequestsInFlight prometheus.Gauge
	activeConnections    prometheus.Gauge
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

		activeConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_active_connections",
				Help: "Number of active HTTP connections",
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

func (m *metrics) SetActiveConnections(count float64) {
	m.activeConnections.Set(count)
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

func (m *metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		defer func() {
			duration := time.Since(start)
			m.RecordRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		}()

		m.IncrementInFlight()
		defer m.DecrementInFlight()

		next.ServeHTTP(wrapped, r)
	})
}

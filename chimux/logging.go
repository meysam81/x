package chimux

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type logRequest struct{ o *options }

func (l *logRequest) shouldSkip(r *http.Request) bool {
	if !l.o.enableHealthzLoging && r.URL.Path == l.o.healthzEndpoint {
		return true
	}

	if !l.o.enableMetricsLoging && r.URL.Path == l.o.metricsEndpoint {
		return true
	}

	return false
}

func (l *logRequest) logWithHeader() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			clientAddr := r.RemoteAddr
			if r.Header.Get("x-forwarded-for") != "" {
				clientAddr = r.Header.Get("x-forwarded-for")
			} else if r.Header.Get("x-real-ip") != "" {
				clientAddr = r.Header.Get("x-real-ip")
			}

			var headers []string
			for header, values := range r.Header {
				headers = append(headers, fmt.Sprintf("%s: %s", strings.ToLower(header), strings.Join(values, "; ")))
			}

			if l.shouldSkip(r) {
				return
			}

			l.o.logger.Info().
				Int("bytes", ww.BytesWritten()).
				Str("duration", time.Since(start).String()).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Str("remote_addr", clientAddr).
				Str("headers", strings.Join(headers, ", ")).
				Send()
		})
	}
}

func (l *logRequest) logWithoutHeader() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			clientAddr := r.RemoteAddr
			if r.Header.Get("x-forwarded-for") != "" {
				clientAddr = r.Header.Get("x-forwarded-for")
			} else if r.Header.Get("x-real-ip") != "" {
				clientAddr = r.Header.Get("x-real-ip")
			}

			if l.shouldSkip(r) {
				return
			}

			l.o.logger.Info().
				Int("bytes", ww.BytesWritten()).
				Str("duration", time.Since(start).String()).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Str("remote_addr", clientAddr).
				Str("user_agent", r.UserAgent()).
				Send()
		})
	}
}

func loggingMiddleware(o *options) func(next http.Handler) http.Handler {
	l := &logRequest{}

	if !o.disableLogHeaders {
		return l.logWithHeader()
	}

	return l.logWithoutHeader()
}

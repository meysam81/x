package chimux

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/meysam81/x/logging"
)

func logWithHeader(logger *logging.Logger) func(next http.Handler) http.Handler {
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

			logger.Info().
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

func logWithoutHeader(logger *logging.Logger) func(next http.Handler) http.Handler {
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

			logger.Info().
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

func loggingMiddleware(logger *logging.Logger, logHeaders bool) func(next http.Handler) http.Handler {
	if logHeaders {
		return logWithHeader(logger)
	}

	return logWithoutHeader(logger)
}

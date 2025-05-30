package logging

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
)

func NetHttpMiddleware() func(next http.Handler) http.Handler {
	l := NewLogger()
	var logger *Logger = &l

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

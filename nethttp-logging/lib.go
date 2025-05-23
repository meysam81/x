package nethttp_logging

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		rw := newResponseWriter(w)

		var reqHeaders []string
		for name, values := range r.Header {
			if strings.EqualFold(name, "Authorization") {
				continue
			}
			for _, value := range values {
				reqHeaders = append(reqHeaders, fmt.Sprintf("%s: %s", name, value))
			}
		}

		next(rw, r)

		var respHeaders []string
		for name, values := range rw.Header() {
			for _, value := range values {
				respHeaders = append(respHeaders, fmt.Sprintf("%s: %s", name, value))
			}
		}

		duration := time.Since(startTime)
		log.Printf(
			"%s %s %s %v %v - %d - %v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			strings.Join(reqHeaders, ", "),
			strings.Join(respHeaders, ", "),
			rw.statusCode,
			duration,
		)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

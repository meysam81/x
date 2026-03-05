package chimux

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/meysam81/x/logging"
)

type headerLogMode int

const (
	headerLogDefault headerLogMode = iota
	headerLogAll
	headerLogNone
)

// defaultLogHeaders is the curated set of headers logged in the default mode.
// Drawn from RFC 9110 (HTTP Semantics), RFC 7239 (Forwarded), W3C Trace Context,
// and common de facto standards (X-Request-Id).
// User-Agent is always logged as a top-level structured field.
var defaultLogHeaders = map[string]struct{}{
	"accept":            {},
	"content-length":    {},
	"content-type":      {},
	"forwarded":         {},
	"host":              {},
	"origin":            {},
	"referer":           {},
	"traceparent":       {},
	"x-forwarded-for":   {},
	"x-forwarded-proto": {},
	"x-request-id":      {},
}

type logRequest struct{ o *options }

func (l *logRequest) shouldSkip(r *http.Request) bool {
	if !l.o.enableHealthzLogging && r.URL.Path == l.o.healthzEndpoint {
		return true
	}

	if !l.o.enableMetricsLogging && r.URL.Path == l.o.metricsEndpoint {
		return true
	}

	return false
}

var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"x-auth-token":        {},
	"x-access-token":      {},
	"authentication":      {},
	"proxy-authorization": {},
}

func isSensitiveHeader(header string) bool {
	_, ok := sensitiveHeaders[strings.ToLower(header)]
	return ok
}

func (l *logRequest) shouldLogHeader(header string) bool {
	switch l.o.headerLogMode {
	case headerLogNone:
		return false
	case headerLogAll:
		return true
	default: // headerLogDefault
		key := strings.ToLower(header)
		if _, ok := defaultLogHeaders[key]; ok {
			return true
		}
		if _, ok := l.o.extraLogHeaders[key]; ok {
			return true
		}
		return false
	}
}

const mask = "TRUNCATED"

func (l *logRequest) log() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			if l.shouldSkip(r) {
				return
			}

			status := ww.Status()

			var event *logging.Event
			switch {
			case status >= 500:
				event = l.o.logger.Error()
			case status >= 400:
				event = l.o.logger.Warn()
			default:
				event = l.o.logger.Info()
			}

			event = event.
				Int("bytes", ww.BytesWritten()).
				Str("duration", time.Since(start).String()).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", status).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent())

			// Skip header iteration when headers are disabled.
			if l.o.headerLogMode != headerLogNone {
				var headers []string
				for header, values := range r.Header {
					if !l.shouldLogHeader(header) {
						continue
					}
					var valueStr string
					if isSensitiveHeader(header) {
						valueStr = mask
					} else {
						valueStr = strings.Join(values, "; ")
					}
					headers = append(headers, fmt.Sprintf("%s: %s", strings.ToLower(header), valueStr))
				}
				if len(headers) > 0 {
					sort.Strings(headers)
					event = event.Str("headers", strings.Join(headers, ", "))
				}
			}

			event.Send()
		})
	}
}

func loggingMiddleware(o *options) func(next http.Handler) http.Handler {
	l := &logRequest{o}
	return l.log()
}

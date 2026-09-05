package chimux

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestReadyzHandler(t *testing.T) {
	tests := []struct {
		name             string
		checks           []ReadyCheck
		wantStatus       int
		wantBodyExact    string
		wantBodyContains []string
	}{
		{
			name:          "no checks",
			wantStatus:    http.StatusOK,
			wantBodyExact: `{"status":"ready"}`,
		},
		{
			name: "two passing checks",
			checks: []ReadyCheck{
				{Name: "db", Check: func(context.Context) error { return nil }},
				{Name: "cache", Check: func(context.Context) error { return nil }},
			},
			wantStatus:    http.StatusOK,
			wantBodyExact: `{"status":"ready"}`,
		},
		{
			name: "one failing check",
			checks: []ReadyCheck{
				{Name: "db", Check: func(context.Context) error { return nil }},
				{Name: "cache", Check: func(context.Context) error { return errors.New("connection refused") }},
			},
			wantStatus:       http.StatusServiceUnavailable,
			wantBodyContains: []string{`"cache"`, "connection refused"},
		},
		{
			name: "panicking check",
			checks: []ReadyCheck{
				{Name: "flaky", Check: func(context.Context) error { panic("boom") }},
			},
			wantStatus:       http.StatusServiceUnavailable,
			wantBodyContains: []string{`"flaky"`, "panic", "boom"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewChi(WithReadyz(tt.checks...))

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}

			body := w.Body.String()
			if tt.wantBodyExact != "" && body != tt.wantBodyExact {
				t.Errorf("body = %q, want %q", body, tt.wantBodyExact)
			}
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(body, want) {
					t.Errorf("body = %q, want substring %q", body, want)
				}
			}
		})
	}
}

func TestReadyzCustomEndpoint(t *testing.T) {
	r := NewChi(WithReadyz(), WithReadyzEndpoint("/internal/readyz"))

	req := httptest.NewRequest(http.MethodGet, "/internal/readyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("custom endpoint status = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("default /readyz status = %d, want 404 once overridden by WithReadyzEndpoint", w.Code)
	}
}

// TestReadyzCheckTimeout verifies a check that ignores context cancellation
// and blocks past the configured timeout is still reported as failed
// promptly, instead of hanging the response for as long as the check runs.
func TestReadyzCheckTimeout(t *testing.T) {
	const (
		timeout  = 50 * time.Millisecond
		blockFor = 2 * time.Second
	)

	r := NewChi(
		WithReadyz(ReadyCheck{
			Name: "slow",
			Check: func(context.Context) error {
				time.Sleep(blockFor) // deliberately ignores ctx
				return nil
			},
		}),
		WithReadyzTimeout(timeout),
	)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body=%s)", w.Code, w.Body.String())
	}
	if elapsed > 2500*time.Millisecond {
		t.Fatalf("handler took %s, want well under 2.5s", elapsed)
	}
	if !strings.Contains(w.Body.String(), `"slow"`) {
		t.Errorf("body = %q, want the timed-out check name", w.Body.String())
	}
}

// TestReadyzExcludedFromAccessLog asserts readyz requests are skipped by the
// access-log middleware the same way healthz and metrics requests are.
func TestReadyzExcludedFromAccessLog(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	r := NewChi(
		WithLoggingMiddleware(),
		WithLogger(&logger),
		WithReadyz(),
	)
	r.Get("/other", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, path := range []string{"/readyz", "/other"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	logs := buf.String()
	if strings.Contains(logs, `"path":"/readyz"`) {
		t.Errorf("readyz request was logged, want it excluded from the access log: %s", logs)
	}
	if !strings.Contains(logs, `"path":"/other"`) {
		t.Errorf("expected /other request to be logged, got: %s", logs)
	}
}

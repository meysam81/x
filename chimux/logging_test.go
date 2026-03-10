package chimux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShouldLogHeader(t *testing.T) {
	tests := []struct {
		name     string
		mode     headerLogMode
		extra    map[string]struct{}
		header   string
		expected bool
	}{
		{"default mode logs content-type", headerLogDefault, nil, "Content-Type", true},
		{"default mode skips user-agent (top-level field)", headerLogDefault, nil, "User-Agent", false},
		{"default mode logs x-request-id", headerLogDefault, nil, "X-Request-Id", true},
		{"default mode logs x-forwarded-for", headerLogDefault, nil, "X-Forwarded-For", true},
		{"default mode skips unknown", headerLogDefault, nil, "X-Custom-Foo", false},
		{"default mode with extra", headerLogDefault, map[string]struct{}{"x-custom-foo": {}}, "X-Custom-Foo", true},
		{"all mode logs everything", headerLogAll, nil, "X-Whatever", true},
		{"none mode logs nothing", headerLogNone, nil, "Content-Type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &logRequest{o: &options{
				headerLogMode:   tt.mode,
				extraLogHeaders: tt.extra,
			}}
			if got := l.shouldLogHeader(tt.header); got != tt.expected {
				t.Errorf("shouldLogHeader(%q) = %v, want %v", tt.header, got, tt.expected)
			}
		})
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		header   string
		expected bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"Cookie", true},
		{"Set-Cookie", true},
		{"Content-Type", false},
		{"X-Api-Key", true},
		{"X-Auth-Token", true},
		{"Proxy-Authorization", true},
		{"User-Agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			if got := isSensitiveHeader(tt.header); got != tt.expected {
				t.Errorf("isSensitiveHeader(%q) = %v, want %v", tt.header, got, tt.expected)
			}
		})
	}
}

func TestLoggingMiddlewareIntegration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("default mode does not panic", func(t *testing.T) {
		r := NewChi(WithLoggingMiddleware())
		r.Get("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Secret-Custom", "should-not-appear")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("all headers mode does not panic", func(t *testing.T) {
		r := NewChi(WithLoggingMiddleware(), WithLogAllHeaders())
		r.Get("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer secret")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("disabled headers mode does not panic", func(t *testing.T) {
		r := NewChi(WithLoggingMiddleware(), WithDisableLogHeaders())
		r.Get("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("extra headers mode does not panic", func(t *testing.T) {
		r := NewChi(WithLoggingMiddleware(), WithLogHeaders("X-Custom-Header"))
		r.Get("/test", handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Custom-Header", "my-value")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}

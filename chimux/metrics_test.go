package chimux

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMetrics drives a single NewChi(WithMetrics()) router through the
// /metrics endpoint it registers. promauto registers collectors on
// Prometheus's global default registry, so a second WithMetrics() router
// built anywhere else in this test binary would panic on duplicate
// registration -- keep this the only one and add subtests instead of new
// top-level tests that build their own.
func TestMetrics(t *testing.T) {
	r := NewChi(WithMetrics())
	r.Get("/api/domains/{domain}/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := httptest.NewServer(r)
	defer srv.Close()

	for _, p := range []string{"/api/domains/a.com/health", "/api/domains/b.com/health", "/does/not/exist"} {
		resp, err := http.Get(srv.URL + p) //nolint:gosec // test server URL
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := http.Get(srv.URL + "/metrics") //nolint:gosec // test server URL
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)

	t.Run("labels requests by route pattern, not raw path", func(t *testing.T) {
		if !strings.Contains(text, `http_requests_total{method="GET",path="/api/domains/{domain}/health",status_code="200"} 2`) {
			t.Errorf("want 2 requests recorded under the route pattern, got:\n%s", text)
		}
		if !strings.Contains(text, `http_requests_total{method="GET",path="unmatched",status_code="404"} 1`) {
			t.Errorf("want 1 unmatched request recorded, got:\n%s", text)
		}
		if strings.Contains(text, `path="/api/domains/a.com/health"`) {
			t.Error("raw path must not appear as a label value")
		}
	})

	t.Run("drops the dead http_active_connections gauge", func(t *testing.T) {
		if !strings.Contains(text, "http_requests_total") {
			t.Error("expected http_requests_total in /metrics output")
		}
		if strings.Contains(text, "http_active_connections") {
			t.Error("http_active_connections must not be exposed; the gauge is never set anywhere")
		}
	})
}

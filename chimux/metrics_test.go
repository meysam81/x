package chimux

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsMiddlewareUsesRoutePattern(t *testing.T) {
	m := newMetrics()
	r := chi.NewRouter()
	r.Use(m.middleware)
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

	if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/api/domains/{domain}/health", "200")); got != 2 {
		t.Fatalf("want 2 requests on the route pattern, got %v", got)
	}
	if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "unmatched", "404")); got != 1 {
		t.Fatalf("want 1 unmatched request, got %v", got)
	}
	if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/api/domains/a.com/health", "200")); got != 0 {
		t.Fatalf("raw path must not be a label value, got %v", got)
	}
}

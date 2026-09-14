package chimux

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestMetrics drives a NewChi(WithMetrics()) router through the /metrics
// endpoint it registers. Metrics are a process-wide singleton (see
// sharedMetrics), so this reads baseline counter values before issuing
// requests and asserts on the delta -- exact absolute values would break
// under `go test -count=N`, where this whole test (and its accumulated
// counters) reruns in the same process.
func TestMetrics(t *testing.T) {
	m := sharedMetrics(nil)

	r := NewChi(WithMetrics(), WithHealthz(), WithReadyz())
	r.Get("/api/domains/{domain}/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := httptest.NewServer(r)
	defer srv.Close()

	matchedBefore := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/api/domains/{domain}/health", "200"))
	unmatchedBefore := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "unmatched", "404"))

	for _, p := range []string{"/api/domains/a.com/health", "/api/domains/b.com/health", "/does/not/exist", "/healthz", "/readyz", "/healthz/", "//readyz"} {
		resp, err := http.Get(srv.URL + p) //nolint:gosec // test server URL
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("labels requests by route pattern, not raw path", func(t *testing.T) {
		if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/api/domains/{domain}/health", "200")) - matchedBefore; got != 2 {
			t.Errorf("want 2 new requests recorded under the route pattern, got %v", got)
		}
		if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "unmatched", "404")) - unmatchedBefore; got != 1 {
			t.Errorf("want 1 new unmatched request recorded, got %v", got)
		}
		if got := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/api/domains/a.com/health", "200")); got != 0 {
			t.Errorf("raw path must not be a label value, got %v", got)
		}
	})

	body := scrapeMetrics(t, srv.URL)

	t.Run("excludes healthz and readyz from metrics recording", func(t *testing.T) {
		if strings.Contains(body, `path="/healthz"`) {
			t.Error("http_requests_total must not have a series for /healthz")
		}
		if strings.Contains(body, `path="/readyz"`) {
			t.Error("http_requests_total must not have a series for /readyz")
		}
		// CleanPath rewrites the route path only; the raw path still reaches
		// the middleware and must be normalised before the skip lookup.
		for _, raw := range []string{`path="/healthz/"`, `path="//readyz"`} {
			if strings.Contains(body, raw) {
				t.Errorf("http_requests_total must not have a series for %s", raw)
			}
		}
	})

	t.Run("drops the dead http_active_connections gauge", func(t *testing.T) {
		if !strings.Contains(body, "http_requests_total") {
			t.Error("expected http_requests_total in /metrics output")
		}
		if strings.Contains(body, "http_active_connections") {
			t.Error("http_active_connections must not be exposed; the gauge is never set anywhere")
		}
	})
}

// TestMetricsSharedAcrossRouters asserts that building metrics into more
// than one router in the same process no longer panics on duplicate
// Prometheus registration (see sharedMetrics), and that both routers'
// traffic lands on the same shared counters.
func TestMetricsSharedAcrossRouters(t *testing.T) {
	m := sharedMetrics(nil)

	r1 := NewChi(WithMetrics())
	r1.Get("/one", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r2 := NewChi(WithMetrics())
	r2.Get("/two", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv1 := httptest.NewServer(r1)
	defer srv1.Close()
	srv2 := httptest.NewServer(r2)
	defer srv2.Close()

	before := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/one", "200")) +
		testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/two", "200"))

	for _, resp := range []string{srv1.URL + "/one", srv2.URL + "/two"} {
		r, err := http.Get(resp) //nolint:gosec // test server URL
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}

	after := testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/one", "200")) +
		testutil.ToFloat64(m.httpRequestsTotal.WithLabelValues("GET", "/two", "200"))

	if got := after - before; got != 2 {
		t.Errorf("want 2 new requests recorded across both routers, got %v", got)
	}
}

func scrapeMetrics(t *testing.T, baseURL string) string {
	t.Helper()

	resp, err := http.Get(baseURL + "/metrics") //nolint:gosec // test server URL
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// TestMetricsBuckets builds the collectors on a private registry (the
// process-wide singleton has already fixed its buckets by the time this
// runs) and checks the histogram carries the requested upper bounds, with
// DefBuckets as the fallback.
func TestMetricsBuckets(t *testing.T) {
	upperBounds := func(t *testing.T, buckets []float64) []float64 {
		t.Helper()
		reg := prometheus.NewRegistry()
		m := newMetrics(reg, buckets)
		m.RecordRequest(http.MethodGet, "/x", http.StatusOK, time.Millisecond)
		families, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range families {
			if f.GetName() != "http_request_duration_seconds" {
				continue
			}
			var got []float64
			for _, b := range f.GetMetric()[0].GetHistogram().GetBucket() {
				got = append(got, b.GetUpperBound())
			}
			return got
		}
		t.Fatal("http_request_duration_seconds not gathered")
		return nil
	}

	t.Run("custom buckets are applied", func(t *testing.T) {
		want := []float64{0.001, 0.005, 0.01}
		if got := upperBounds(t, want); !slices.Equal(got, want) {
			t.Errorf("upper bounds = %v, want %v", got, want)
		}
	})

	t.Run("nil falls back to DefBuckets", func(t *testing.T) {
		if got := upperBounds(t, nil); !slices.Equal(got, prometheus.DefBuckets) {
			t.Errorf("upper bounds = %v, want DefBuckets %v", got, prometheus.DefBuckets)
		}
	})
}

package tracing

import (
	"context"
	"testing"

	"github.com/meysam81/x/logging"
)

// TestNewTracer guards against regressing the "conflicting Schema URL"
// bug: tracing.go's own resource.NewWithAttributes call used to pull in
// semconv v1.34.0 while otel/sdk's resource.Default() used the schema baked
// into v1.39.0, so resource.Merge failed for every enabled tracer. Both
// sides now import v1.39.0.
func TestNewTracer(t *testing.T) {
	logger := logging.NewLogger()

	t.Run("enabled with a configured endpoint succeeds without network access", func(t *testing.T) {
		// Exporter construction is lazy (otlptracehttp.New does not dial),
		// so this never touches the network despite naming an endpoint.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tr, err := NewTracer(ctx, &TracingConfig{
			Enabled:         true,
			ServiceName:     "t",
			OTLPEndpointURL: "http://127.0.0.1:1",
		}, &logger)
		if err != nil {
			t.Fatalf("NewTracer() error = %v, want nil", err)
		}
		if tr == nil {
			t.Fatal("NewTracer() returned a nil tracer")
		}
		if tr.Tracer == nil {
			t.Error("enabled tracer should have a non-nil Tracer")
		}
		if tr.Provider == nil {
			t.Error("enabled tracer should have a non-nil Provider")
		}
	})

	t.Run("disabled returns a no-op tracer", func(t *testing.T) {
		tr, err := NewTracer(context.Background(), &TracingConfig{Enabled: false}, &logger)
		if err != nil {
			t.Fatalf("NewTracer() error = %v, want nil", err)
		}
		if tr == nil {
			t.Fatal("NewTracer() returned a nil tracer")
		}
		if tr.Tracer != nil {
			t.Error("disabled tracer should have a nil Tracer")
		}
		if tr.Provider != nil {
			t.Error("disabled tracer should have a nil Provider")
		}
		if got := tr.GetTracer(); got == nil {
			t.Error("GetTracer() returned nil for a disabled tracer")
		}
	})
}

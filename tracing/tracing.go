// Package tracing provides OpenTelemetry tracing setup with OTLP HTTP export,
// automatic shutdown on context cancellation, and chi-compatible HTTP middleware.
package tracing

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/meysam81/x/logging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-chi/chi/v5/middleware"
)

// TracingConfig holds configuration for the OpenTelemetry tracer.
type TracingConfig struct {
	ServiceName     string
	ServiceVersion  string
	OTLPEndpointURL string
	Enabled         bool
	ServerName      string

	ShutdownTimeoutSec int
}

// Tracer wraps an OpenTelemetry TracerProvider with convenience methods.
type Tracer struct {
	Provider *sdktrace.TracerProvider
	Tracer   trace.Tracer
	Logger   *logging.Logger
	Config   *TracingConfig
}

// NewTracer creates a Tracer configured with an OTLP HTTP exporter. It returns
// a no-op tracer if Enabled is false. The provider is automatically shut down
// when ctx is cancelled.
func NewTracer(ctx context.Context, config *TracingConfig, logger *logging.Logger) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{Logger: logger}, nil
	}

	opts := []otlptracehttp.Option{}

	if config.OTLPEndpointURL != "" {
		otlptracehttp.WithEndpointURL(config.OTLPEndpointURL)
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	serverName := config.ServerName
	if serverName == "" {
		serverName = hostname
	}

	resource, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.ServerAddress(serverName),
			semconv.HostName(hostname),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	t := &Tracer{
		Provider: provider,
		Tracer:   tracer,
		Logger:   logger,
		Config:   config,
	}

	if config.ShutdownTimeoutSec == 0 {
		config.ShutdownTimeoutSec = 10
	}
	ctxT, cancelT := context.WithTimeout(context.Background(), time.Duration(config.ShutdownTimeoutSec)*time.Second)
	go func() {
		defer cancelT()
		<-ctx.Done()
		if err := t.Shutdown(ctxT); err != nil {
			logger.Error().Err(err).Msg("failed shutting down the tracer")
		}
	}()

	return t, nil
}

// GetTracer returns the underlying tracer, or a no-op tracer if tracing is disabled.
func (t *Tracer) GetTracer() trace.Tracer {
	if t.Tracer == nil {
		return otel.GetTracerProvider().Tracer("noop")
	}
	return t.Tracer
}

// Shutdown flushes pending spans and shuts down the trace provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.Provider == nil {
		return nil
	}
	return t.Provider.Shutdown(ctx)
}

// StartSpan starts a new span; it returns the original context unchanged if tracing is disabled.
func (t *Tracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	if t.Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.Tracer.Start(ctx, spanName)
}

// DetachSpanFromContext returns a new background context carrying only the span
// from ctx. Use this to continue tracing in goroutines that outlive the original
// request context.
func (t *Tracer) DetachSpanFromContext(ctx context.Context) context.Context {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return context.Background()
	}
	return trace.ContextWithSpan(context.Background(), span)
}

// HTTPMiddleware is a chi-compatible middleware that creates a server span per
// request with standard HTTP semantic conventions.
func (t *Tracer) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t.Tracer == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := t.Tracer.Start(ctx, r.Method+" "+r.URL.Path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			span.SetAttributes(
				semconv.HTTPResponseStatusCode(ww.Status()),
			)
		}()

		span.SetAttributes(
			semconv.ServiceName(t.Config.ServiceName),
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.HTTPRoute(r.URL.Path),
			semconv.URLPath(r.URL.Path),
			semconv.URLQuery(r.URL.RawQuery),
			semconv.URLScheme(r.URL.Scheme),
			semconv.ServerAddress(r.Host),
			semconv.UserAgentOriginal(r.UserAgent()),
			semconv.NetworkProtocolName("http"),
			semconv.NetworkProtocolVersion(r.Proto),
		)

		r = r.WithContext(ctx)
		next.ServeHTTP(ww, r)
	})
}

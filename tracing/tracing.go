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

type TracingConfig struct {
	ServiceName     string
	ServiceVersion  string
	OTLPEndpointURL string
	Enabled         bool
	ServerName      string
}

type Tracer struct {
	Provider *sdktrace.TracerProvider
	Tracer   trace.Tracer
	Logger   *logging.Logger
	Config   *TracingConfig
}

func NewTracer(ctx context.Context, config *TracingConfig, logger *logging.Logger) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{Logger: logger}, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(config.OTLPEndpointURL),
	)
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

	ctxT, cancelT := context.WithTimeout(context.Background(), 5*time.Second)
	go func() {
		defer cancelT()
		<-ctx.Done()
		if err := t.Shutdown(ctxT); err != nil {
			logger.Error().Err(err).Msg("failed shutting down the tracer")
		}
	}()

	return t, nil
}

func (t *Tracer) GetTracer() trace.Tracer {
	if t.Tracer == nil {
		return otel.GetTracerProvider().Tracer("noop")
	}
	return t.Tracer
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.Provider == nil {
		return nil
	}
	return t.Provider.Shutdown(ctx)
}

func (t *Tracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	if t.Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.Tracer.Start(ctx, spanName)
}

func (t *Tracer) DetachSpanFromContext(ctx context.Context) context.Context {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return context.Background()
	}
	return trace.ContextWithSpan(context.Background(), span)
}

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

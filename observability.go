package morondanga

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// initObservability sets up the global OTLP tracer and log providers when
// observability is enabled in the service config.
func (s *Service) initObservability() error {
	obs := s.Configuration().GetObservability()
	if obs == nil || !obs.Enabled {
		s.otelShutdown = func() {}
		s.tracer = otel.Tracer(s.Configuration().GetApp().Name)
		return nil
	}

	endpoint := obs.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(s.Configuration().GetApp().Name),
		),
	)
	if err != nil {
		return fmt.Errorf("otel resource: %w", err)
	}

	traceExp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint+"/v1/traces"),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("otel trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logExp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(endpoint+"/v1/logs"),
		otlploghttp.WithInsecure(),
		otlploghttp.WithTimeout(5*time.Second),
	)
	if err != nil {
		_ = tp.Shutdown(context.Background())
		return fmt.Errorf("otel log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	s.tracer = otel.Tracer(s.Configuration().GetApp().Name)
	s.otelShutdown = func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutCtx)
		_ = lp.Shutdown(shutCtx)
	}

	return nil
}

// Tracer returns the service's OpenTelemetry tracer.
// When observability is disabled, it returns a no-op tracer.
func (s *Service) Tracer() trace.Tracer {
	return s.tracer
}

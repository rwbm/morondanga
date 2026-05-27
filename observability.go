package morondanga

import (
	"context"
	"fmt"
	"runtime"
	"time"

	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
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

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(endpoint + "/v1/traces"),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(5 * time.Second),
	}
	if obs.ApiKey != "" {
		traceOpts = append(traceOpts, otlptracehttp.WithHeaders(map[string]string{"X-API-Key": obs.ApiKey}))
	}
	traceExp, err := otlptracehttp.New(ctx, traceOpts...)
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

	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(endpoint + "/v1/metrics"),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithTimeout(5 * time.Second),
	}
	if obs.ApiKey != "" {
		metricOpts = append(metricOpts, otlpmetrichttp.WithHeaders(map[string]string{"X-API-Key": obs.ApiKey}))
	}
	metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		_ = tp.Shutdown(context.Background())
		return fmt.Errorf("otel metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(30*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	logOpts := []otlploghttp.Option{
		otlploghttp.WithEndpointURL(endpoint + "/v1/logs"),
		otlploghttp.WithInsecure(),
		otlploghttp.WithTimeout(5 * time.Second),
	}
	if obs.ApiKey != "" {
		logOpts = append(logOpts, otlploghttp.WithHeaders(map[string]string{"X-API-Key": obs.ApiKey}))
	}
	logExp, err := otlploghttp.New(ctx, logOpts...)
	if err != nil {
		_ = tp.Shutdown(context.Background())
		_ = mp.Shutdown(context.Background())
		return fmt.Errorf("otel log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	// Go runtime metrics (goroutines, GC, etc.) via contrib package.
	_ = runtimemetrics.Start(runtimemetrics.WithMinimumReadMemStatsInterval(30 * time.Second))

	// process.memory.usage — standard OTLP semconv, backed by runtime.MemStats.Sys.
	meter := mp.Meter(s.Configuration().GetApp().Name)
	memGauge, _ := meter.Int64ObservableGauge("process.memory.usage",
		metric.WithUnit("By"),
		metric.WithDescription("Process virtual memory reserved from the OS"),
	)
	_, _ = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		o.ObserveInt64(memGauge, int64(ms.Sys))
		return nil
	}, memGauge)

	s.tracer = otel.Tracer(s.Configuration().GetApp().Name)
	s.otelShutdown = func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = tp.Shutdown(shutCtx)
		_ = mp.Shutdown(shutCtx)
		_ = lp.Shutdown(shutCtx)
	}

	return nil
}

// Tracer returns the service's OpenTelemetry tracer.
// When observability is disabled, it returns a no-op tracer.
func (s *Service) Tracer() trace.Tracer {
	return s.tracer
}

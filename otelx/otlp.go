package otelx

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func SetupOTLP(t *Tracer, tracerName string) (trace.Tracer, error) {
	ctx := context.Background()

	clientOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(t.Config.Providers.OTLP.ServerURL),
	}

	if t.Config.Providers.OTLP.Insecure {
		clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
	}

	exp, err := otlptrace.New(
		ctx, otlptracehttp.NewClient(clientOpts...),
	)
	if err != nil {
		return nil, err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(t.Config.ServiceName),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(
			t.Config.Providers.OTLP.Sampling.SamplingRatio,
		))),
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	return tp.Tracer(tracerName), nil
}

// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"

	"go.opentelemetry.io/contrib/propagators/b3"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func SetupOTLP(t *Tracer, tracerName string, c *Config) (trace.Tracer, error) {
	ctx := context.Background()

	var client otlptrace.Client
	if c.Providers.OTLP.Transport == "GRPC" {
		client = getOTLPGRPClient(c)
	} else {
		client = getOTLPHttpClient(c)
	}

	exp, err := otlptrace.New(
		ctx, client,
	)
	if err != nil {
		return nil, err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(c.ServiceName),
			semconv.DeploymentEnvironmentKey.String(c.DeploymentEnvironment),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(
			c.Providers.OTLP.Sampling.SamplingRatio,
		))),
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		jaegerPropagator.Jaeger{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)),
		propagation.Baggage{},
	))

	return tp.Tracer(tracerName), nil
}

func getOTLPGRPClient(c *Config) otlptrace.Client {
	clientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(c.Providers.OTLP.ServerURL),
	}

	if c.Providers.OTLP.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	if c.Providers.OTLP.AuthorizationHeader != "" {
		clientOpts = append(clientOpts,
			otlptracegrpc.WithHeaders(map[string]string{"Authorization": c.Providers.OTLP.AuthorizationHeader}),
		)
	}

	return otlptracegrpc.NewClient(clientOpts...)
}

func getOTLPHttpClient(c *Config) otlptrace.Client {
	clientOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(c.Providers.OTLP.ServerURL),
	}

	if c.Providers.OTLP.Insecure {
		clientOpts = append(clientOpts, otlptracehttp.WithInsecure())
	}

	if c.Providers.OTLP.AuthorizationHeader != "" {
		clientOpts = append(clientOpts,
			otlptracehttp.WithHeaders(map[string]string{"Authorization": c.Providers.OTLP.AuthorizationHeader}),
		)
	}

	return otlptracehttp.NewClient(clientOpts...)
}

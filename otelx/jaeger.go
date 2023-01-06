// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"net"

	"go.opentelemetry.io/contrib/propagators/b3"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

// Optionally, Config.Providers.Jaeger.LocalAgentAddress can be set.
// NOTE: If Config.Providers.Jaeger.Sampling.ServerURL is not specfied,
// AlwaysSample is used.
func SetupJaeger(t *Tracer, tracerName string, c *Config) (trace.Tracer, error) {
	host, port, err := net.SplitHostPort(c.Providers.Jaeger.LocalAgentAddress)
	if err != nil {
		return nil, err
	}

	exp, err := jaeger.New(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(host), jaeger.WithAgentPort(port),
		),
	)
	if err != nil {
		return nil, err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(c.ServiceName),
		)),
	}

	samplingServerURL := c.Providers.Jaeger.Sampling.ServerURL

	if samplingServerURL != "" {
		jaegerRemoteSampler := jaegerremote.New(
			"jaegerremote",
			jaegerremote.WithSamplingServerURL(samplingServerURL),
			jaegerremote.WithInitialSampler(sdktrace.TraceIDRatioBased(c.Providers.Jaeger.Sampling.TraceIdRatio)),
		)
		tpOpts = append(tpOpts, sdktrace.WithSampler(jaegerRemoteSampler))
	} else {
		tpOpts = append(tpOpts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	// At the moment, software across our cloud stack only support Zipkin (B3)
	// and Jaeger propagation formats. Proposals for standardized formats for
	// context propagation are in the works (ref: https://www.w3.org/TR/trace-context/
	// and https://www.w3.org/TR/baggage/).
	//
	// Simply add propagation.TraceContext{} and propagation.Baggage{}
	// here to enable those as well.
	prop := propagation.NewCompositeTextMapPropagator(
		jaegerPropagator.Jaeger{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)),
	)
	otel.SetTextMapPropagator(prop)
	return tp.Tracer(tracerName), nil
}

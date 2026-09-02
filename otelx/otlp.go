// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"net/url"
	"strings"

	"go.opentelemetry.io/contrib/propagators/b3"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const otlpTracesPath = "/v1/traces"

func otlpHTTPOptions(c *OTLPConfig) []otlptracehttp.Option {
	opts := make([]otlptracehttp.Option, 0, 4)
	insecure := c.Insecure

	if u, err := url.Parse(c.ServerURL); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(u.Host))
		if p := otlpTracesURLPath(u.Path); p != "" {
			opts = append(opts, otlptracehttp.WithURLPath(p))
		}
		insecure = u.Scheme == "http"
	} else {
		opts = append(opts, otlptracehttp.WithEndpoint(c.ServerURL))
	}

	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if c.AuthorizationHeader != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{"Authorization": c.AuthorizationHeader}))
	}

	return opts
}

// otlpTracesURLPath maps a URL path to the OTLP HTTP traces path.
// Empty or "/" leaves the exporter default (/v1/traces). Other paths are treated
// as a base prefix (OTEL_EXPORTER_OTLP_ENDPOINT semantics) unless they already
// end with /v1/traces.
func otlpTracesURLPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return ""
	}
	p = strings.TrimRight(p, "/")
	if strings.HasSuffix(p, otlpTracesPath) {
		return p
	}
	return p + otlpTracesPath
}

func SetupOTLP(t *Tracer, tracerName string, c *Config) (trace.Tracer, error) {
	ctx := context.Background()

	clientOpts := otlpHTTPOptions(&c.Providers.OTLP)

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
			semconv.ServiceName(c.ServiceName),
			semconv.DeploymentEnvironmentName(c.DeploymentEnvironment),
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

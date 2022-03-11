package otelx

import (
	"fmt"
	"io"

	"github.com/ory/x/logrusx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	Config *Config

	l      *logrusx.Logger
	tracer trace.Tracer
	closer io.Closer
}

// Creates a new tracer. If name is empty, a default tracer name is used
// instead. See: https://godocs.io/go.opentelemetry.io/otel/sdk/trace#TracerProvider.Tracer
func New(name string, l *logrusx.Logger, c *Config) (*Tracer, error) {
	t := &Tracer{Config: c, l: l}

	if err := t.setup(name); err != nil {
		return nil, err
	}

	return t, nil
}

// setup configures a Resource and sets up the TracerProvider
// to send spans to the provided URL.
//
// Endpoint configuration is implicitly read from the below environment
// variables, by default:
//
//    OTEL_EXPORTER_JAEGER_AGENT_HOST
//    OTEL_EXPORTER_JAEGER_AGENT_PORT
//
// Optionally, Config.Providers.Jaeger.LocalAgentAddress can be set.
// NOTE: the default sampling ratio is set to 0.5. You might want to change
// this in production.
func (t *Tracer) setup(name string) error {
	switch t.Config.Provider {
	case "jaeger":
		exp, err := jaeger.New(jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(t.Config.Providers.Jaeger.LocalAgentHost),
			jaeger.WithAgentPort(fmt.Sprint(t.Config.Providers.Jaeger.LocalAgentPort)),
		))
		if err != nil {
			return err
		}

		samplingRatio := t.Config.Providers.Jaeger.SamplingRatio
		if samplingRatio == 0 {
			samplingRatio = 0.5
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(t.Config.ServiceName),
			)),
			sdktrace.WithSampler(sdktrace.TraceIDRatioBased(
				samplingRatio,
			)),
		)

		otel.SetTracerProvider(tp)
		t.tracer = tp.Tracer(name)
	default:
		return errors.Errorf("unknown tracer: %s", t.Config.Provider)
	}
	return nil
}

// IsLoaded returns true if the tracer has been loaded.
func (t *Tracer) IsLoaded() bool {
	if t == nil || t.tracer == nil {
		return false
	}
	return true
}

// Returns the wrapped tracer.
func (t *Tracer) Tracer() trace.Tracer {
	return t.tracer
}

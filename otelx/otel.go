package otelx

import (
	"fmt"
	"io"
	"os"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/stringsx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/propagation/b3"
	jaegerPropagator "go.opentelemetry.io/contrib/propagation/jaeger"
	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
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
		host := stringsx.Coalesce(
			t.Config.Providers.Jaeger.LocalAgentHost,
			os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_HOST"),
		)
		var port string
		if t.Config.Providers.Jaeger.LocalAgentPort != 0 {
			port = t.Config.Providers.Jaeger.LocalAgentHost
		} else {
			port = os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_PORT")
		}

		exp, err = jaeger.New(
			jaeger.WithAgentEndpoint(
				jaeger.WithAgentHost(host), jaeger.WithAgentPort(port),
			),
		)
		if err != nil {
			return err
		}

		tpOpts := []sdktrace.TracerProviderOption{
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(t.Config.ServiceName),
			)),
		}

		if t.Config.Providers.Jaeger.Sampling.ServerURL != "" {
			jaegerRemoteSampler := jaegerremote.New(
				"jaegerremote",
				jaegerremote.WithSamplingServerURL(t.Config.Providers.Jaeger.Sampling.ServerURL),
			)
			tpOpts = append(tpOpts, sdktrace.WithSampler(jaegerRemoteSampler))
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

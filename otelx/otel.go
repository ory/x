package otelx

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/stringsx"
)

type Tracer struct {
	Config *Config

	l      *logrusx.Logger
	tracer trace.Tracer
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

// Creates a new no-op tracer.
func NewNoop(l *logrusx.Logger, c *Config) *Tracer {
	tp := trace.NewNoopTracerProvider()
	t := &Tracer{Config: c, l: l, tracer: tp.Tracer("")}
	return t
}

// setup sets up the tracer.
func (t *Tracer) setup(name string) error {
	switch f := stringsx.SwitchExact(t.Config.Provider); {
	case f.AddCase("jaeger"):
		tracer, err := SetupJaeger(t, name)
		if err != nil {
			return err
		}

		t.tracer = tracer
		t.l.Infof("Jaeger tracer configured! Sending spans to %s", t.Config.Providers.Jaeger.LocalAgentAddress)
	case f.AddCase("zipkin"):
		tracer, err := SetupZipkin(t, name)
		if err != nil {
			return err
		}

		t.tracer = tracer
		t.l.Infof("Zipkin tracer configured! Sending spans to %s", t.Config.Providers.Zipkin.ServerURL)
	case f.AddCase("otel"):
		tracer, err := SetupOTLP(t, name)
		if err != nil {
			return err
		}

		t.tracer = tracer
		t.l.Infof("OTLP tracer configured! Sending spans to %s", t.Config.Providers.OTLP.ServerURL)
	case f.AddCase(""):
		t.l.Infof("No tracer configured - skipping tracing setup")
	default:
		return f.ToUnknownCaseErr()
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

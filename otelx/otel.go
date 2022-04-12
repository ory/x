package otelx

import (
	"github.com/ory/x/logrusx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
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
	switch t.Config.Provider {
	case "jaeger":
		tracer, err := SetupJaeger(t, name)
		if err != nil {
			return err
		}

		t.tracer = tracer
		t.l.Infof("Jaeger tracer configured!")
	case "":
		t.l.Infof("No tracer configured - skipping tracing setup")
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

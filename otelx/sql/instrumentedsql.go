package sql

import (
	"context"
	"database/sql/driver"

	"github.com/luna-duclos/instrumentedsql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type tracer struct {
}

type span struct {
	tracer
	ctx    context.Context
	parent trace.Span
}

func NewTracer() instrumentedsql.Tracer { return tracer{} }

// GetSpan returns a span
func (t tracer) GetSpan(ctx context.Context) instrumentedsql.Span {
	if ctx == nil {
		return span{ctx: nil}
	}

	return span{parent: trace.SpanFromContext(ctx), tracer: t}
}

func (s span) NewChild(name string) instrumentedsql.Span {
	if s.ctx == nil {
		return s
	}

	var parent trace.Span
	tp := otel.GetTracerProvider().Tracer("github.com/ory/x/otelx/sql")
	// if s.parent == nil {
	// 	_, parent = tp.Start(context.Background(), name)
	// 	return span{parent: parent, tracer: s.tracer}
	// } else {
	_, parent = tp.Start(s.ctx, name)

	return span{ctx: s.ctx, parent: parent}
}

func (s span) SetLabel(k, v string) {
	if s.parent == nil {
		return
	}
	s.parent.SetAttributes(attribute.String(k, v))
}

func (s span) SetError(err error) {
	if err == nil || err == driver.ErrSkip {
		return
	}

	if s.parent == nil {
		return
	}

	s.parent.SetStatus(codes.Error, err.Error())
	s.parent.AddEvent("error", trace.WithAttributes(
		attribute.String("message", err.Error())),
	)
}

func (s span) Finish() {
	if s.parent == nil {
		return
	}
	s.parent.End()
}

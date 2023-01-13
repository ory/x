// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var errPanic = errors.New("panic-error")

func TestWithSpan(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	assert.NoError(t, WithSpan(ctx, "no-error", func(ctx context.Context) error { return nil }))
	assert.Error(t, WithSpan(ctx, "error", func(ctx context.Context) error { return errors.New("some-error") }))
	assert.PanicsWithError(t, errPanic.Error(), func() {
		WithSpan(ctx, "panic", func(ctx context.Context) error {
			panic(errPanic)
		})
	})
	assert.PanicsWithValue(t, errPanic, func() {
		WithSpan(ctx, "panic", func(ctx context.Context) error {
			panic(errPanic)
		})
	})
	assert.PanicsWithValue(t, "panic-string", func() {
		WithSpan(ctx, "panic", func(ctx context.Context) error {
			panic("panic-string")
		})
	})
}

func returnsNormally(ctx context.Context) (err error) {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "returnsNormally")
	defer End(span, &err)
	return nil
}

func returnsError(ctx context.Context) (err error) {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "returnsError")
	defer End(span, &err)
	return errors.New("error from returnsError()")
}

func returnsNamedError(ctx context.Context) (err error) {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "returnsNamedError")
	defer End(span, &err)
	err2 := errors.New("err2 message")
	return err2
}

func panics(ctx context.Context) (err error) {
	_, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "panics")
	defer End(span, &err)
	panic(errors.New("panic from panics()"))
}

func TestEnd(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracer := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)).Tracer("test")
	ctx, span := tracer.Start(context.Background(), "parent")
	defer span.End()

	assert.NoError(t, returnsNormally(ctx))
	require.NotEmpty(t, recorder.Ended())
	assert.Equal(t, last(recorder).Name(), "returnsNormally")
	assert.Equal(t, last(recorder).Status(), sdktrace.Status{codes.Unset, ""})

	assert.Errorf(t, returnsError(ctx), "error from returnsError()")
	require.NotEmpty(t, recorder.Ended())
	assert.Equal(t, last(recorder).Name(), "returnsError")
	assert.Equal(t, last(recorder).Status(), sdktrace.Status{codes.Error, "error from returnsError()"})

	assert.Errorf(t, returnsNamedError(ctx), "err2 message")
	require.NotEmpty(t, recorder.Ended())
	assert.Equal(t, last(recorder).Name(), "returnsNamedError")
	assert.Equal(t, last(recorder).Status(), sdktrace.Status{codes.Error, "err2 message"})

	assert.PanicsWithError(t, "panic from panics()", func() { panics(ctx) })
	require.NotEmpty(t, recorder.Ended())
	assert.Equal(t, last(recorder).Name(), "panics")
	assert.Equal(t, last(recorder).Status(), sdktrace.Status{codes.Error, "panic: panic from panics()"})

	span.End()
	require.NotEmpty(t, recorder.Ended())
	assert.Equal(t, last(recorder).Name(), "parent")
	assert.Equal(t, last(recorder).Status(), sdktrace.Status{codes.Unset, ""})
}

func last(r *tracetest.SpanRecorder) sdktrace.ReadOnlySpan {
	ended := r.Ended()
	if len(ended) == 0 {
		return nil
	}
	return ended[len(ended)-1]
}

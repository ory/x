// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"errors"
	"testing"

	"github.com/instana/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

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

var errPanic = errors.New("panic-error")

// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// WithSpan wraps execution of f in a span identified by name.
//
// If f returns an error or panics, the span status will be set to the error
// state. The error (or panic) will be propagated unmodified.
//
// f will be wrapped in a child span by default. To make a new root span
// instead, pass the trace.WithNewRoot() option.
func WithSpan(ctx context.Context, name string, f func(context.Context) error, opts ...trace.SpanStartOption) (err error) {
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, name, opts...)
	defer func() {
		defer span.End()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				span.SetStatus(codes.Error, "panic: "+e.Error())
			case interface{ String() string }:
				span.SetStatus(codes.Error, "panic: "+e.String())
			case string:
				span.SetStatus(codes.Error, "panic: "+e)
			default:
				span.SetStatus(codes.Error, "panic")
			}
			panic(r)
		}
	}()
	return f(ctx)
}

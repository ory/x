// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"fmt"

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
		if r := recover(); r != nil {
			setErrorStatusPanic(span, r)
			panic(r)
		} else if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}()
	return f(ctx)
}

// End finishes span, and automatically sets the error state if *err is not nil
// or during panicking.
//
// Usage:
//
//	func Divide(ctx context.Context, numerator, denominator int) (ratio int, err error) {
//		ctx, span := tracer.Start(ctx, "my-operation")
//		defer otelx.End(span, &err)
//		if denominator == 0 {
//			return 0, errors.New("cannot divide by zero")
//		}
//		return numerator / denominator, nil
//	}
func End(span trace.Span, err *error) {
	defer span.End()
	if r := recover(); r != nil {
		setErrorStatusPanic(span, r)
		panic(r)
	}
	if err == nil || *err == nil {
		return
	}
	span.SetStatus(codes.Error, (*err).Error())
}

func setErrorStatusPanic(span trace.Span, recovered any) {
	switch e := recovered.(type) {
	case error, string, fmt.Stringer:
		span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", e))
	default:
		span.SetStatus(codes.Error, "panic")
	case nil:
		// nothing
	}
}

// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	pkgerrors "github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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
			setErrorTags(span, err)
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
//		ctx, span := tracer.Start(ctx, "Divide")
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
	setErrorTags(span, *err)
}

func setErrorStatusPanic(span trace.Span, recovered any) {
	span.SetAttributes(semconv.ExceptionEscaped(true))
	if t := reflect.TypeOf(recovered); t != nil {
		span.SetAttributes(semconv.ExceptionType(t.String()))
	}
	switch e := recovered.(type) {
	case error:
		span.SetStatus(codes.Error, "panic: "+e.Error())
		setErrorTags(span, e)
	case string, fmt.Stringer:
		span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", e))
	default:
		span.SetStatus(codes.Error, "panic")
	case nil:
		// nothing
	}
}

func setErrorTags(span trace.Span, err error) {
	span.SetAttributes(
		attribute.String("error", err.Error()),
		attribute.String("error.message", err.Error()),                        // compat
		attribute.String("error.type", fmt.Sprintf("%T", errors.Unwrap(err))), // the innermost error type is the most useful here
	)
	if e := interface{ StackTrace() pkgerrors.StackTrace }(nil); errors.As(err, &e) {
		span.SetAttributes(attribute.String("error.stack", fmt.Sprintf("%+v", e.StackTrace())))
	}
	if e := interface{ Reason() string }(nil); errors.As(err, &e) {
		span.SetAttributes(attribute.String("error.reason", e.Reason()))
	}
	if e := interface{ Debug() string }(nil); errors.As(err, &e) {
		span.SetAttributes(attribute.String("error.debug", e.Debug()))
	}
	if e := interface{ ID() string }(nil); errors.As(err, &e) {
		span.SetAttributes(attribute.String("error.id", e.ID()))
	}
	if e := interface{ Details() map[string]interface{} }(nil); errors.As(err, &e) {
		for k, v := range e.Details() {
			span.SetAttributes(attribute.String("error.details."+k, fmt.Sprintf("%v", v)))
		}
	}
}

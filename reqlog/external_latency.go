// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package reqlog

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// ExternalCallsMiddleware is a middleware that sets up the request context to measure external calls.
// It has to be used before any other middleware that reads the final external latency.
func ExternalCallsMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	container := contextContainer{
		latencies: make([]externalLatency, 0),
	}
	next(rw, r.WithContext(
		context.WithValue(r.Context(), internalLatencyKey, &container),
	))
}

// MeasureExternalCall measures the duration of a function and records it as an external call.
// The wrapped function's return value is returned.
func MeasureExternalCall[T any](ctx context.Context, cause, detail string, f func() T) T {
	defer StartMeasureExternalCall(ctx, cause, detail)()
	return f()
}

// MeasureExternalCallErr measures the duration of a function and records it as an external call.
// The wrapped function's return value and error is returned.
func MeasureExternalCallErr[T any](ctx context.Context, cause, detail string, f func() (T, error)) (T, error) {
	defer StartMeasureExternalCall(ctx, cause, detail)()
	return f()
}

// StartMeasureExternalCall starts measuring the duration of an external call.
// The returned function has to be called to record the duration.
func StartMeasureExternalCall(ctx context.Context, cause, detail string) func() {
	container, ok := ctx.Value(internalLatencyKey).(*contextContainer)
	if !ok {
		return func() {}
	}

	start := time.Now()
	return func() {
		container.Lock()
		defer container.Unlock()
		container.latencies = append(container.latencies, externalLatency{
			Took:   time.Since(start),
			Cause:  cause,
			Detail: detail,
		})
	}
}

// TotalExternalLatency returns the total duration of all external calls.
func TotalExternalLatency(ctx context.Context) (total time.Duration) {
	if _, ok := ctx.Value(disableExternalLatencyMeasurement).(bool); ok {
		return 0
	}
	container, ok := ctx.Value(internalLatencyKey).(*contextContainer)
	if !ok {
		return 0
	}

	container.Lock()
	defer container.Unlock()
	for _, l := range container.latencies {
		total += l.Took
	}
	return total
}

// WithDisableExternalLatencyMeasurement returns a context that does not measure external latencies.
// Use this when you want to disable external latency measurements for a specific request.
func WithDisableExternalLatencyMeasurement(ctx context.Context) context.Context {
	return context.WithValue(ctx, disableExternalLatencyMeasurement, true)
}

type (
	externalLatency = struct {
		Took          time.Duration
		Cause, Detail string
	}
	contextContainer = struct {
		latencies []externalLatency
		sync.Mutex
	}
	contextKey int
)

const (
	internalLatencyKey                contextKey = 1
	disableExternalLatencyMeasurement contextKey = 2
)

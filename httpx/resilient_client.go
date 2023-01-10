// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/ory/x/logrusx"
)

type resilientOptions struct {
	ctx                  context.Context
	c                    *http.Client
	l                    interface{}
	retryWaitMin         time.Duration
	retryWaitMax         time.Duration
	retryMax             int
	noInternalIPs        bool
	internalIPExceptions []string
	tracer               trace.Tracer
}

func newResilientOptions() *resilientOptions {
	connTimeout := time.Minute
	return &resilientOptions{
		c:            &http.Client{Timeout: connTimeout},
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 30 * time.Second,
		retryMax:     4,
		l:            log.New(io.Discard, "", log.LstdFlags),
	}
}

// ResilientOptions is a set of options for the ResilientClient.
type ResilientOptions func(o *resilientOptions)

// ResilientClientWithClient sets the underlying http client to use.
func ResilientClientWithClient(c *http.Client) ResilientOptions {
	return func(o *resilientOptions) {
		o.c = c
	}
}

// ResilientClientWithTracer wraps the http clients transport with a tracing instrumentation
func ResilientClientWithTracer(tracer trace.Tracer) ResilientOptions {
	return func(o *resilientOptions) {
		o.tracer = tracer
	}
}

// ResilientClientWithMaxRetry sets the maximum number of retries.
func ResilientClientWithMaxRetry(retryMax int) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryMax = retryMax
	}
}

// ResilientClientWithMinxRetryWait sets the minimum wait time between retries.
func ResilientClientWithMinxRetryWait(retryWaitMin time.Duration) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryWaitMin = retryWaitMin
	}
}

// ResilientClientWithMaxRetryWait sets the maximum wait time for a retry.
func ResilientClientWithMaxRetryWait(retryWaitMax time.Duration) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryWaitMax = retryWaitMax
	}
}

// ResilientClientWithConnectionTimeout sets the connection timeout for the client.
func ResilientClientWithConnectionTimeout(connTimeout time.Duration) ResilientOptions {
	return func(o *resilientOptions) {
		o.c.Timeout = connTimeout
	}
}

// ResilientClientWithLogger sets the logger to be used by the client.
func ResilientClientWithLogger(l *logrusx.Logger) ResilientOptions {
	return func(o *resilientOptions) {
		o.l = l
	}
}

// ResilientClientDisallowInternalIPs disallows internal IPs from being used.
func ResilientClientDisallowInternalIPs() ResilientOptions {
	return func(o *resilientOptions) {
		o.noInternalIPs = true
	}
}

// ResilientClientAllowInternalIPRequestsTo allows requests to the exact matching URLs even
// if they are internal IPs.
func ResilientClientAllowInternalIPRequestsTo(urls ...string) ResilientOptions {
	return func(o *resilientOptions) {
		o.internalIPExceptions = urls
	}
}

// NewResilientClient creates a new ResilientClient.
func NewResilientClient(opts ...ResilientOptions) *retryablehttp.Client {
	o := newResilientOptions()
	for _, f := range opts {
		f(o)
	}

	if o.noInternalIPs == true {
		o.c.Transport = &NoInternalIPRoundTripper{
			RoundTripper:         o.c.Transport,
			internalIPExceptions: o.internalIPExceptions,
		}
	}

	if o.tracer != nil {
		o.c.Transport = otelhttp.NewTransport(o.c.Transport)
	}

	return &retryablehttp.Client{
		HTTPClient:   o.c,
		Logger:       o.l,
		RetryWaitMin: o.retryWaitMin,
		RetryWaitMax: o.retryWaitMax,
		RetryMax:     o.retryMax,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}
}

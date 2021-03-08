package httpx

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"

	"github.com/ory/x/logrusx"
)

type resilientOptions struct {
	ctx          context.Context
	c            *http.Client
	l            interface{}
	retryWaitMin time.Duration
	retryWaitMax time.Duration
	retryMax     int
}

func newResilientOptions() *resilientOptions {
	return &resilientOptions{
		c:            &http.Client{Timeout: time.Minute},
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 30 * time.Second,
		retryMax:     4,
		l:            log.New(io.Discard, "", log.LstdFlags),
	}
}

type ResilientOptions func(o *resilientOptions)

func ResilientClientWithClient(c *http.Client) ResilientOptions {
	return func(o *resilientOptions) {
		o.c = c
	}
}

func ResilientClientWithMaxRetry(retryMax int) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryMax = retryMax
	}
}

func ResilientClientWithMinxRetryWait(retryWaitMin time.Duration) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryWaitMin = retryWaitMin
	}
}

func ResilientClientWithMaxRetryWait(retryWaitMax time.Duration) ResilientOptions {
	return func(o *resilientOptions) {
		o.retryWaitMax = retryWaitMax
	}
}

func ResilientClientWithLogger(l *logrusx.Logger) ResilientOptions {
	return func(o *resilientOptions) {
		o.l = l
	}
}

func NewResilientClient(opts ...ResilientOptions) *retryablehttp.Client {
	var o resilientOptions
	for _, f := range opts {
		f(&o)
	}

	return &retryablehttp.Client{
		HTTPClient:   cleanhttp.DefaultPooledClient(),
		Logger:       o.l,
		RetryWaitMin: o.retryWaitMin,
		RetryWaitMax: o.retryWaitMax,
		RetryMax:     o.retryMax,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}
}

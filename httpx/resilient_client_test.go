// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoPrivateIPs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Hello, world!"))
	}))
	t.Cleanup(ts.Close)

	target, err := url.ParseRequestURI(ts.URL)
	require.NoError(t, err)

	_, port, err := net.SplitHostPort(target.Host)
	require.NoError(t, err)

	allowed := "http://localhost:" + port + "/foobar"

	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientDisallowInternalIPs(),
		ResilientClientAllowInternalIPRequestsTo(allowed),
	)

	for destination, passes := range map[string]bool{
		"http://127.0.0.1:" + port:             false,
		"http://localhost:" + port:             false,
		"http://192.168.178.5:" + port:         false,
		allowed:                                true,
		"http://localhost:" + port + "/FOOBAR": false,
	} {
		_, err := c.Get(destination)
		if !passes {
			require.Error(t, err)
			assert.Contains(t, err.Error(), "is not a public IP address")
		} else {
			require.NoError(t, err)
		}
	}
}

var errClient = &http.Client{Transport: errRoundTripper{}}

func TestNoPrivateIPsRespectsWrappedClient(t *testing.T) {
	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientDisallowInternalIPs(),
		ResilientClientWithClient(errClient),
	)
	_, err := c.Get("https://google.com")
	require.ErrorIs(t, err, fakeErr)
}

func TestClientWithTracerRespectsWrappedClient(t *testing.T) {
	tracer := otel.Tracer("github.com/ory/x/httpx test")
	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientWithTracer(tracer),
		ResilientClientWithClient(errClient),
	)
	_, err := c.Get("https://google.com")
	require.ErrorIs(t, err, fakeErr)
}

func TestClientWithMultiConfigRespectsWrapperClient(t *testing.T) {
	tracer := otel.Tracer("github.com/ory/x/httpx test")
	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientWithTracer(tracer),
		ResilientClientDisallowInternalIPs(),
		ResilientClientWithClient(errClient),
	)
	_, err := c.Get("https://google.com")
	require.ErrorIs(t, err, fakeErr)
}

func TestClientWithTracer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("Hello, world!"))
	}))
	t.Cleanup(ts.Close)

	tracer := otel.Tracer("github.com/ory/x/httpx test")
	c := NewResilientClient(
		ResilientClientWithTracer(tracer),
	)

	target, err := url.ParseRequestURI(ts.URL)
	require.NoError(t, err)

	_, err = c.Get(target.String())

	assert.NoError(t, err)
}

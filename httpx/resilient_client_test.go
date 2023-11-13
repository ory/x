// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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

	allowedURL := "http://localhost:" + port + "/foobar"
	allowedGlob := "http://localhost:" + port + "/glob/*"

	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientDisallowInternalIPs(),
		ResilientClientAllowInternalIPRequestsTo(allowedURL, allowedGlob),
	)

	for i := 0; i < 10; i++ {
		for destination, passes := range map[string]bool{
			"http://127.0.0.1:" + port:                   false,
			"http://localhost:" + port:                   false,
			"http://192.168.178.5:" + port:               false,
			allowedURL:                                   true,
			"http://localhost:" + port + "/glob/bar":     true,
			"http://localhost:" + port + "/glob/bar/baz": false,
			"http://localhost:" + port + "/FOOBAR":       false,
		} {
			_, err := c.Get(destination)
			if !passes {
				require.Errorf(t, err, "dest = %s", destination)
				assert.Containsf(t, err.Error(), "is not a permitted destination", "dest = %s", destination)
			} else {
				require.NoErrorf(t, err, "dest = %s", destination)
			}
		}
	}
}

// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"code.dny.dev/ssrf"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivateIPs(t *testing.T) {
	testCases := []struct {
		url                 string
		disallowInternalIPs bool
		allowedIP           bool
	}{
		{
			url:                 "http://127.0.0.1/foobar",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://localhost/foobar",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://127.0.0.1:56789/test",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://192.168.178.5:56789",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://127.0.0.1:56789/foobar",
			disallowInternalIPs: true,
			allowedIP:           true,
		},
		{
			url:                 "http://127.0.0.1:56789/glob/bar",
			disallowInternalIPs: true,
			allowedIP:           true,
		},
		{
			url:                 "http://127.0.0.1:56789/glob/bar/baz",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://127.0.0.1:56789/FOOBAR",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://100.64.1.1:80/private",
			disallowInternalIPs: true,
			allowedIP:           true,
		},
		{
			url:                 "http://100.64.1.1:80/route",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://198.18.99.99/forbidden",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			// Even if in the allowed requests, no exceptions can be made.
			url:                 "http://198.18.99.99/allowed",
			disallowInternalIPs: true,
			allowedIP:           false,
		},
		{
			url:                 "http://127.0.0.1",
			disallowInternalIPs: false,
			allowedIP:           true,
		},
		{
			url:                 "http://192.168.178.5",
			disallowInternalIPs: false,
			allowedIP:           true,
		},
		{
			url:                 "http://127.0.0.1:80/glob/bar",
			disallowInternalIPs: false,
			allowedIP:           true,
		},
		{
			url:                 "http://100.64.1.1:80/route",
			disallowInternalIPs: false,
			allowedIP:           true,
		},
	}
	for _, tt := range testCases {
		t.Run(
			fmt.Sprintf("%s should be allowed %v when disallowed internal IPs is %v", tt.url, tt.allowedIP, tt.disallowInternalIPs),
			func(t *testing.T) {
				options := []ResilientOptions{
					ResilientClientWithMaxRetry(0),
					ResilientClientWithConnectionTimeout(50 * time.Millisecond),
				}
				if tt.disallowInternalIPs {
					options = append(options, ResilientClientDisallowInternalIPs())
					options = append(options, ResilientClientAllowInternalIPRequestsTo(
						"http://127.0.0.1:56789/foobar",
						"http://127.0.0.1:56789/glob/*",
						"http://100.64.1.1:80/private",
						"http://198.18.99.99/allowed"))
				}

				c := NewResilientClient(options...)
				_, err := c.Get(tt.url)
				if tt.allowedIP {
					assert.NotErrorIs(t, err, ssrf.ErrProhibitedIP)
				} else {
					assert.ErrorIs(t, err, ssrf.ErrProhibitedIP)
				}
			})
	}
}

func TestNoIPV6(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    *retryablehttp.Client
	}{
		{
			"internal IPs allowed",
			NewResilientClient(
				ResilientClientWithMaxRetry(1),
				ResilientClientNoIPv6(),
			),
		}, {
			"internal IPs disallowed",
			NewResilientClient(
				ResilientClientWithMaxRetry(1),
				ResilientClientDisallowInternalIPs(),
				ResilientClientNoIPv6(),
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var connectDone int32
			ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
				DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
					for _, ip := range dnsInfo.Addrs {
						netIP, ok := netip.AddrFromSlice(ip.IP)
						assert.True(t, ok)
						assert.Truef(t, netIP.Is4(), "ip = %s", ip)
					}
				},
				ConnectDone: func(network, addr string, err error) {
					atomic.AddInt32(&connectDone, 1)
					assert.NoError(t, err)
					assert.Equalf(t, "tcp4", network, "network = %s addr = %s", network, addr)
				},
			})

			// Dual stack
			req, err := retryablehttp.NewRequestWithContext(ctx, "GET", "http://dual.tlund.se/", nil)
			require.NoError(t, err)
			atomic.StoreInt32(&connectDone, 0)
			res, err := tc.c.Do(req)
			require.GreaterOrEqual(t, int32(1), atomic.LoadInt32(&connectDone))
			require.NoError(t, err)
			t.Cleanup(func() { _ = res.Body.Close() })
			require.EqualValues(t, http.StatusOK, res.StatusCode)

			// IPv4 only
			req, err = retryablehttp.NewRequestWithContext(ctx, "GET", "http://ipv4.tlund.se/", nil)
			require.NoError(t, err)
			atomic.StoreInt32(&connectDone, 0)
			res, err = tc.c.Do(req)
			require.EqualValues(t, 1, atomic.LoadInt32(&connectDone))
			require.NoError(t, err)
			t.Cleanup(func() { _ = res.Body.Close() })
			require.EqualValues(t, http.StatusOK, res.StatusCode)

			// IPv6 only
			req, err = retryablehttp.NewRequestWithContext(ctx, "GET", "http://ipv6.tlund.se/", nil)
			require.NoError(t, err)
			atomic.StoreInt32(&connectDone, 0)
			_, err = tc.c.Do(req)
			require.EqualValues(t, 0, atomic.LoadInt32(&connectDone))
			require.ErrorContains(t, err, "no such host")
		})
	}
}

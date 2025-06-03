// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"time"

	"code.dny.dev/ssrf"
	"github.com/gobwas/glob"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var _ http.RoundTripper = (*noInternalIPRoundTripper)(nil)

type noInternalIPRoundTripper struct {
	onWhitelist, notOnWhitelist http.RoundTripper
	internalIPExceptions        []string
}

// NewNoInternalIPRoundTripper creates a RoundTripper that disallows
// non-publicly routable IP addresses, except for URLs matching the given
// exception globs.
// Deprecated: Use ResilientClientDisallowInternalIPs instead.
func NewNoInternalIPRoundTripper(exceptions []string) http.RoundTripper {
	return &noInternalIPRoundTripper{
		onWhitelist:          allowInternalAllowIPv6,
		notOnWhitelist:       prohibitInternalAllowIPv6,
		internalIPExceptions: exceptions,
	}
}

// RoundTrip implements http.RoundTripper.
func (n noInternalIPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	incoming := IncomingRequestURL(request)
	incoming.RawQuery = ""
	incoming.RawFragment = ""
	for _, exception := range n.internalIPExceptions {
		compiled, err := glob.Compile(exception, '.', '/')
		if err != nil {
			return nil, err
		}
		if compiled.Match(incoming.String()) {
			return n.onWhitelist.RoundTrip(request)
		}
	}

	return n.notOnWhitelist.RoundTrip(request)
}

var (
	prohibitInternalAllowIPv6    http.RoundTripper
	prohibitInternalProhibitIPv6 http.RoundTripper
	allowInternalAllowIPv6       http.RoundTripper
	allowInternalProhibitIPv6    http.RoundTripper
)

func init() {
	t, d := newDefaultTransport()
	d.Control = ssrf.New(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4", "tcp6"),
	).Safe
	prohibitInternalAllowIPv6 = otelTransport(t)
}

func init() {
	t, d := newDefaultTransport()
	d.Control = ssrf.New(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4"),
	).Safe
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return d.DialContext(ctx, "tcp4", addr)
	}
	prohibitInternalProhibitIPv6 = otelTransport(t)
}

func init() {
	allowedV4Prefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),     // Private-Use (RFC 1918)
		netip.MustParsePrefix("127.0.0.0/8"),    // Loopback (RFC 1122, Section 3.2.1.3))
		netip.MustParsePrefix("169.254.0.0/16"), // Link Local (RFC 3927)
		netip.MustParsePrefix("172.16.0.0/12"),  // Private-Use (RFC 1918)
		netip.MustParsePrefix("192.168.0.0/16"), // Private-Use (RFC 1918)
	}

	t, d := newDefaultTransport()
	d.Control = ssrf.New(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4", "tcp6"),
		ssrf.WithAllowedV4Prefixes(allowedV4Prefixes...),
		ssrf.WithAllowedV6Prefixes(append(
			[]netip.Prefix{
				netip.MustParsePrefix("::1/128"),  // Loopback (RFC 4193)
				netip.MustParsePrefix("fc00::/7"), // Unique Local (RFC 4193)
			}, mustConvertToNAT64Prefixes(allowedV4Prefixes)...,
		)...),
	).Safe
	allowInternalAllowIPv6 = otelTransport(t)
}

func init() {
	allowedV4Prefixes := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),     // Private-Use (RFC 1918)
		netip.MustParsePrefix("127.0.0.0/8"),    // Loopback (RFC 1122, Section 3.2.1.3))
		netip.MustParsePrefix("169.254.0.0/16"), // Link Local (RFC 3927)
		netip.MustParsePrefix("172.16.0.0/12"),  // Private-Use (RFC 1918)
		netip.MustParsePrefix("192.168.0.0/16"), // Private-Use (RFC 1918)
	}

	t, d := newDefaultTransport()
	d.Control = ssrf.New(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4"),
		ssrf.WithAllowedV4Prefixes(allowedV4Prefixes...),
		ssrf.WithAllowedV6Prefixes(append(
			[]netip.Prefix{
				netip.MustParsePrefix("::1/128"),  // Loopback (RFC 4193)
				netip.MustParsePrefix("fc00::/7"), // Unique Local (RFC 4193)
			}, mustConvertToNAT64Prefixes(allowedV4Prefixes)...,
		)...),
	).Safe
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return d.DialContext(ctx, "tcp4", addr)
	}
	allowInternalProhibitIPv6 = otelTransport(t)
}

func newDefaultTransport() (*http.Transport, *net.Dialer) {
	dialer := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}, &dialer
}

func otelTransport(t *http.Transport) http.RoundTripper {
	return otelhttp.NewTransport(t, otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
		return otelhttptrace.NewClientTrace(ctx, otelhttptrace.WithoutHeaders(), otelhttptrace.WithoutSubSpans())
	}))
}

func mustConvertToNAT64Prefixes(ps []netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, len(ps))
	for i, p := range ps {
		out[i] = mustConvertToNAT64Prefix(p)
	}
	return out
}

func mustConvertToNAT64Prefix(p netip.Prefix) netip.Prefix {
	var nat64Base = netip.MustParsePrefix("64:ff9b::/96") // NAT64 Prefix (RFC 6052)

	if !p.Addr().Is4() {
		panic(fmt.Errorf("prefix %v is not an IPv4 prefix", p))
	}

	ipv4Len := p.Bits()
	if ipv4Len > 32 {
		panic(fmt.Errorf("invalid IPv4 prefix length: %d", ipv4Len))
	}

	newLen := 96 + ipv4Len
	ip4 := p.Addr().As4()

	baseBytes := nat64Base.Addr().As16()
	copy(baseBytes[12:], ip4[:])

	return netip.PrefixFrom(netip.AddrFrom16(baseBytes), newLen)
}

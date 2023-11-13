// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"time"

	"code.dny.dev/ssrf"
	"github.com/gobwas/glob"
)

var _ http.RoundTripper = (*noInternalIPRoundTripper)(nil)

type noInternalIPRoundTripper struct {
	onWhitelist, notOnWhitelist http.RoundTripper
	internalIPExceptions        []string
}

// NewNoInternalIPRoundTripper creates a RoundTripper that disallows
// non-publicly routable IP addresses, except for URLs matching the given
// exception globs.
func NewNoInternalIPRoundTripper(exceptions []string) http.RoundTripper {
	if len(exceptions) > 0 {
		prohibitInternal := newSSRFTransport(ssrf.New(
			ssrf.WithAnyPort(),
			ssrf.WithNetworks("tcp4", "tcp6"),
		))

		allowInternal := newSSRFTransport(ssrf.New(
			ssrf.WithAnyPort(),
			ssrf.WithNetworks("tcp4", "tcp6"),
			ssrf.WithAllowedV4Prefixes(
				netip.MustParsePrefix("10.0.0.0/8"),     // Private-Use (RFC 1918)
				netip.MustParsePrefix("127.0.0.0/8"),    // Loopback (RFC 1122, Section 3.2.1.3))
				netip.MustParsePrefix("169.254.0.0/16"), // Link Local (RFC 3927)
				netip.MustParsePrefix("172.16.0.0/12"),  // Private-Use (RFC 1918)
				netip.MustParsePrefix("192.168.0.0/16"), // Private-Use (RFC 1918)
			),
			ssrf.WithAllowedV6Prefixes(
				netip.MustParsePrefix("::1/128"),  // Loopback (RFC 4193)
				netip.MustParsePrefix("fc00::/7"), // Unique Local (RFC 4193)
			),
		))
		return noInternalIPRoundTripper{
			onWhitelist:          allowInternal,
			notOnWhitelist:       prohibitInternal,
			internalIPExceptions: exceptions,
		}
	}
	prohibitInternal := newSSRFTransport(ssrf.New(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4", "tcp6"),
	))
	return noInternalIPRoundTripper{
		onWhitelist:    prohibitInternal,
		notOnWhitelist: prohibitInternal,
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

func newSSRFTransport(g *ssrf.Guardian) http.RoundTripper {
	t := newDefaultTransport()
	t.DialContext = (&net.Dialer{Control: g.Safe}).DialContext
	return t
}

func newDefaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

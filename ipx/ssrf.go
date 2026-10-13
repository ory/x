// Copyright © 2026 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package ipx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"code.dny.dev/ssrf"
)

var (
	ProhibitInternalDialFunc = ssrfDialFunc(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4", "tcp6"),
	)

	AllowInternalDialFunc = ssrfDialFunc(
		ssrf.WithAnyPort(),
		ssrf.WithNetworks("tcp4", "tcp6"),
		ssrf.WithAllowedV4Prefixes(allowedInternalV4Prefixes...),
		ssrf.WithAllowedV6Prefixes(allowedInternalV6Prefixes...),
	)
)

// AllowedInternalPrefixesEnv names the environment variable that replaces the
// prefixes AllowInternalDialFunc may dial. It takes a comma-separated list of
// IPv4 and IPv6 prefixes, e.g. "10.0.0.0/8,100.64.0.0/10,fd00::/8". If it is
// unset or empty, the defaults below apply. An invalid prefix panics when the
// package is initialized.
const AllowedInternalPrefixesEnv = "ORY_SSRF_ALLOWED_INTERNAL_PREFIXES"

var (
	defaultAllowedInternalV4Prefixes = []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),     // Private-Use (RFC 1918)
		netip.MustParsePrefix("127.0.0.0/8"),    // Loopback (RFC 1122, Section 3.2.1.3))
		netip.MustParsePrefix("169.254.0.0/16"), // Link Local (RFC 3927)
		netip.MustParsePrefix("172.16.0.0/12"),  // Private-Use (RFC 1918)
		netip.MustParsePrefix("192.168.0.0/16"), // Private-Use (RFC 1918)
	}
	defaultAllowedInternalV6Prefixes = []netip.Prefix{
		netip.MustParsePrefix("::1/128"),  // Loopback (RFC 4193)
		netip.MustParsePrefix("fc00::/7"), // Unique Local (RFC 4193)
	}

	allowedInternalV4Prefixes, allowedInternalV6Prefixes = allowedInternalPrefixes(os.Getenv(AllowedInternalPrefixesEnv))
)

func allowedInternalPrefixes(env string) (v4, v6 []netip.Prefix) {
	if strings.TrimSpace(env) == "" {
		return defaultAllowedInternalV4Prefixes, defaultAllowedInternalV6Prefixes
	}
	for _, s := range strings.Split(env, ",") {
		p, err := netip.ParsePrefix(strings.TrimSpace(s))
		if err == nil && p.Addr().Is4In6() {
			err = fmt.Errorf("%s is IPv4-mapped and never matches, use the IPv4 form", p)
		}
		if err != nil {
			panic(fmt.Sprintf("%s: %v", AllowedInternalPrefixesEnv, err))
		}
		if p.Addr().Is4() {
			v4 = append(v4, p.Masked())
		} else {
			v6 = append(v6, p.Masked())
		}
	}
	return v4, v6
}

func ssrfDialFunc(opt ...ssrf.Option) func(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	dialer.Control = ssrf.New(opt...).Safe
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, err := dialer.DialContext(ctx, network, address)
		if err == nil {
			return c, nil
		}

		if dnsErr, ok := errors.AsType[*net.DNSError](err); ok {
			// Copy the `*net.DNSError` before masking `Server` to avoid a data
			// race: the DNS resolver uses `singleflight` to deduplicate
			// concurrent lookups, so multiple goroutines may receive the same
			// `*net.DNSError` pointer. Mutating it in place races with concurrent
			// readers (e.g. the `otelhttp` `dnsDone` trace hook).
			maskedDNS := *dnsErr
			maskedDNS.Server = "" // Mask our DNS server's IP address.
			// Also copy the outer `*net.OpError` (if present) to preserve the
			// full error chain that callers depend on, replacing its `Err` field
			// with the masked copy.
			// Surprisingly Go does not have a good way of manipulating error chains.
			if opErr, ok := errors.AsType[*net.OpError](err); ok {
				maskedOp := *opErr
				maskedOp.Err = &maskedDNS
				return nil, &maskedOp
			}
			return nil, &maskedDNS
		}

		if !errors.Is(err, ssrf.ErrProhibitedIP) {
			return nil, err
		}

		host, _, _ := net.SplitHostPort(address)
		_, addrErr := netip.ParseAddrPort(address)
		if addrErr != nil {
			// We were given a DNS name: the error we return must look like a DNS error.
			return nil, &net.OpError{
				Op:   "dial",
				Net:  network,
				Addr: nil,
				Err: &net.DNSError{
					Err:         "no such host",
					Name:        host,
					Server:      "",
					IsTimeout:   false,
					IsTemporary: false,
					IsNotFound:  true,
				},
			}
		}
		return nil, &net.OpError{
			Op:   "dial",
			Net:  network,
			Addr: nil,
			Err:  errors.New("no route to host"),
		}
	}
}

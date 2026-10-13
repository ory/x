// Copyright © 2026 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package ipx

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"code.dny.dev/ssrf"
)

func allowedInternal(v4, v6 []netip.Prefix, ip string) bool {
	addr := netip.MustParseAddr(ip)
	network := "tcp6"
	if addr.Is4() {
		network = "tcp4"
	}
	g := ssrf.New(ssrf.WithAnyPort(), ssrf.WithNetworks("tcp4", "tcp6"), ssrf.WithAllowedV4Prefixes(v4...), ssrf.WithAllowedV6Prefixes(v6...))
	return g.Safe(network, netip.AddrPortFrom(addr, 80).String(), nil) == nil
}

// refused reports whether err is the error ssrfDialFunc returns for a prohibited IP.
func refused(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Addr == nil && opErr.Err.Error() == "no route to host"
}

func TestAllowedInternalPrefixes(t *testing.T) {
	for _, tc := range []struct {
		env     string
		allowed []string
		denied  []string
	}{
		{
			env:     "",
			allowed: []string{"10.1.2.3", "127.0.0.1", "169.254.169.254", "172.16.0.1", "192.168.1.1", "1.1.1.1", "::1", "fd00::1"},
			denied:  []string{"100.64.0.1", "198.18.0.1", "0.0.0.1", "240.0.0.1", "::", "fe80::1", "::ffff:a9fe:a9fe"},
		},
		{
			env:     "  ",
			allowed: []string{"169.254.169.254"},
			denied:  []string{"198.18.0.1"},
		},
		{
			env:     "10.0.0.0/8, 100.64.0.0/10,198.18.0.0/15 ,fd00::/8",
			allowed: []string{"10.1.2.3", "100.64.0.1", "198.18.0.1", "fd00::1", "1.1.1.1", "2606:4700::1111"},
			denied:  []string{"169.254.169.254", "172.16.0.1", "127.0.0.1", "::1", "fc00::1"},
		},
		{
			env:     "198.18.1.2/15",
			allowed: []string{"198.18.0.1", "198.19.255.254"},
			denied:  []string{"10.1.2.3"},
		},
	} {
		v4, v6 := allowedInternalPrefixes(tc.env)
		for _, ip := range tc.allowed {
			if !allowedInternal(v4, v6, ip) {
				t.Errorf("env %q: %s must be allowed", tc.env, ip)
			}
		}
		for _, ip := range tc.denied {
			if allowedInternal(v4, v6, ip) {
				t.Errorf("env %q: %s must be denied", tc.env, ip)
			}
		}
	}
}

func TestAllowedInternalPrefixesInvalid(t *testing.T) {
	for _, env := range []string{"10.0.0.0/8,nope", "10.0.0.0/8,", "10.0.0.1", "::ffff:198.18.0.0/111"} {
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(r.(string), AllowedInternalPrefixesEnv) {
					t.Errorf("env %q: want a panic naming %s, got %v", env, AllowedInternalPrefixesEnv, r)
				}
			}()
			allowedInternalPrefixes(env)
		}()
	}
}

func TestAllowInternalDialFunc(t *testing.T) {
	if os.Getenv(AllowedInternalPrefixesEnv) != "" {
		t.Skipf("%s is set", AllowedInternalPrefixesEnv)
	}
	for addr, wantRefused := range map[string]bool{
		"198.18.0.1:9": true,
		"127.0.0.1:1":  false,
	} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		conn, err := AllowInternalDialFunc(ctx, "tcp", addr)
		cancel()
		if conn != nil {
			_ = conn.Close()
		}
		if refused(err) != wantRefused {
			t.Errorf("%s: want refused=%v, got %v", addr, wantRefused, err)
		}
	}
}

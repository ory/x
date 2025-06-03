// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package ipx

import (
	"net/netip"
	"testing"

	"code.dny.dev/ssrf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustConvertToNAT64Prefix_ValidInputs(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"192.0.2.0/24", "64:ff9b::c000:200/120"},
		{"10.0.0.0/8", "64:ff9b::a00:0/104"},
		{"0.0.0.0/0", "64:ff9b::/96"},
	}

	for _, tc := range tests {
		p := netip.MustParsePrefix(tc.in)
		got := MustConvertToNAT64Prefix(p)
		assert.Equal(t, tc.want, got.String(), "MustConvertToNAT64Prefix(%q)", tc.in)
	}
}

func TestMustConvertToNAT64Prefix_PanicsOnInvalid(t *testing.T) {
	assert.Panics(t, func() {
		MustConvertToNAT64Prefix(netip.MustParsePrefix("2001:db8::/32"))
	}, "Expected panic for non-IPv4 prefix")
}

func TestMustConvertToNAT64Prefixes_SliceConversion(t *testing.T) {
	input := []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("10.1.0.0/16"),
	}
	want := []string{
		"64:ff9b::c000:200/120",
		"64:ff9b::a01:0/112",
	}

	got := MustConvertToNAT64Prefixes(input)
	require.Len(t, got, len(want), "MustConvertToNAT64Prefixes returned %d entries; want %d", len(got), len(want))
	for i, p := range got {
		assert.Equal(t, want[i], p.String(), "Element %d", i)
	}
}

func TestPublicIPv4Nat64Prefixes_BasicSanity(t *testing.T) {
	out := PublicIPv4Nat64Prefixes()
	require.NotEmpty(t, out, "Expected nonempty slice")

	for _, p := range out {
		s := p.String()
		assert.True(t, p.Addr().Is6(), "Returned prefix %q is not IPv6", s)
		assert.GreaterOrEqual(t, p.Bits(), 96, "Unexpected prefix length for %q", s)
		assert.LessOrEqual(t, p.Bits(), 128, "Unexpected prefix length for %q", s)
		base := ssrf.IPv6NAT64Prefix.Addr().String()[:len("64:ff9b:")]
		assert.True(t, len(s) >= len(base) && s[:len(base)] == base,
			"Prefix %q does not start with NAT64 base", s,
		)
	}
}

func TestComplementIPv4_FullCoverage(t *testing.T) {
	denied := ssrf.IPv4DeniedPrefixes
	allowed := complementIPv4(denied)

	// 1) No denied overlaps any allowed
	for _, d := range denied {
		dStart, dEnd := prefixRange(d)
		for _, a := range allowed {
			aStart, aEnd := prefixRange(a)
			overlaps := !(dEnd < aStart || aEnd < dStart)
			assert.False(t, overlaps, "Denied prefix %q overlaps allowed prefix %q", d, a)
		}
	}

	// 2) Denied + allowed cover all /8 blocks exactly once
	seen := make(map[string]struct{})
	for _, set := range [][]netip.Prefix{denied, allowed} {
		for _, p := range set {
			start, end := prefixRange(p)
			for ip := start; ip <= end; {
				octet := byte((ip >> 24) & 0xFF)
				key, err := netip.AddrFrom4([4]byte{octet, 0, 0, 0}).Prefix(8)
				require.NoError(t, err)
				seen[key.String()] = struct{}{}
				ip += 1 << 24
				if ip == 0 {
					break
				}
			}
		}
	}
	assert.Len(t, seen, 256, "Denied+allowed did not cover all 256 /8 blocks; covered %d", len(seen))
}

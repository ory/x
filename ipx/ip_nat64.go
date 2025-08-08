// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package ipx

import (
	"fmt"
	"net/netip"

	"code.dny.dev/ssrf"
)

// PublicIPv4Nat64Prefixes returns the list of public IPv4 to their NAT64 (RFC 6052) IPv6 representation
func PublicIPv4Nat64Prefixes() []netip.Prefix {
	return MustConvertToNAT64Prefixes(complementIPv4(ssrf.IPv4DeniedPrefixes))
}

// MustConvertToNAT64Prefixes convert a list of IPv4 prefixes to a NAT64 (RFC 6052) list of IPv6 prefixes or panic
func MustConvertToNAT64Prefixes(ps []netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, len(ps))
	for i, p := range ps {
		out[i] = MustConvertToNAT64Prefix(p)
	}
	return out
}

// MustConvertToNAT64Prefix convert an IPv4 prefix to a NAT64 (RFC 6052) IPv6 prefix or panic
func MustConvertToNAT64Prefix(p netip.Prefix) netip.Prefix {
	if !p.Addr().Is4() {
		panic(fmt.Errorf("prefix %v is not an IPv4 prefix", p))
	}

	ipv4Len := p.Bits()
	if ipv4Len > 32 {
		panic(fmt.Errorf("invalid IPv4 prefix length: %d", ipv4Len))
	}

	newLen := 96 + ipv4Len
	ip4 := p.Addr().As4()

	baseBytes := ssrf.IPv6NAT64Prefix.Addr().As16()
	copy(baseBytes[12:], ip4[:])

	return netip.PrefixFrom(netip.AddrFrom16(baseBytes), newLen)
}

// ipToUint32 converts a netip.Addr (IPv4) to its uint32 representation.
func ipToUint32(a netip.Addr) uint32 {
	b := a.As4()
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// prefixRange returns the first and last IPv4 addresses of p as uint32.
func prefixRange(p netip.Prefix) (start, end uint32) {
	// Masked() ensures the address is the network address.
	base := ipToUint32(p.Masked().Addr())
	ones, _ := p.Bits(), p.Addr().BitLen()
	size := uint32(1) << (32 - ones)
	return base, base + size - 1
}

// subtractPrefix(r, p) returns the set of IPv4 prefixes covering (r − p)
// without using AddrRange().
func subtractPrefix(r, p netip.Prefix) []netip.Prefix {
	rStart, rEnd := prefixRange(r)
	pStart, pEnd := prefixRange(p)

	// No overlap
	if rEnd < pStart || pEnd < rStart {
		return []netip.Prefix{r}
	}
	// p covers r completely
	if pStart <= rStart && rEnd <= pEnd {
		return nil
	}
	// r strictly contains p: split r into two / (r.Bits()+1) children.
	childLen := r.Bits() + 1
	c1 := netip.PrefixFrom(r.Addr(), childLen)

	// Compute base for second child.
	increment := uint32(1) << (32 - childLen)
	raw := ipToUint32(r.Addr()) + increment
	b0 := byte((raw >> 24) & 0xFF)
	b1 := byte((raw >> 16) & 0xFF)
	b2 := byte((raw >> 8) & 0xFF)
	b3 := byte(raw & 0xFF)
	c2 := netip.PrefixFrom(netip.AddrFrom4([4]byte{b0, b1, b2, b3}), childLen)

	out := subtractPrefix(c1, p)
	out = append(out, subtractPrefix(c2, p)...)
	return out
}

// complementIPv4 returns the minimal set of IPv4 prefixes covering all
// addresses in 0.0.0.0/0 that are not in any of the input prefixes.
func complementIPv4(input []netip.Prefix) []netip.Prefix {
	remainder := []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}
	for _, p := range input {
		next := make([]netip.Prefix, 0, len(remainder))
		for _, r := range remainder {
			next = append(next, subtractPrefix(r, p)...)
		}
		remainder = next
	}
	return remainder
}

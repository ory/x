// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/ory/x/stringsx"

	"github.com/pkg/errors"
)

// ErrPrivateIPAddressDisallowed is returned when a private IP address is disallowed.
type ErrPrivateIPAddressDisallowed error

// DisallowPrivateIPAddressesWhenSet is a wrapper for DisallowIPPrivateAddresses which returns valid
// when ipOrHostnameOrURL is empty.
func DisallowPrivateIPAddressesWhenSet(ipOrHostnameOrURL string) error {
	if ipOrHostnameOrURL == "" {
		return nil
	}
	return DisallowIPPrivateAddresses(ipOrHostnameOrURL)
}

// DisallowIPPrivateAddresses returns nil for a domain (with NS lookup), IP, or IPv6 address if it
// does not resolve to a private IP subnet. This is a first level of defense against
// SSRF attacks by disallowing any domain or IP to resolve to a private network range.
//
// Please keep in mind that validations for domains is valid only when looking up.
// A malicious actor could easily update the DSN record post validation to point
// to an internal IP
func DisallowIPPrivateAddresses(ipOrHostnameOrURL string) error {
	lookup := func(hostname string) ([]net.IP, error) {
		lookup, err := net.LookupIP(hostname)
		if err != nil {
			if dnsErr := new(net.DNSError); errors.As(err, &dnsErr) && (dnsErr.IsNotFound || dnsErr.IsTemporary) {
				// If the hostname does not resolve, we can't validate it. So yeah,
				// I guess we're allowing it.
				return nil, nil
			}
			return nil, errors.WithStack(err)
		}
		return lookup, nil
	}

	var ips []net.IP
	ip := net.ParseIP(ipOrHostnameOrURL)
	if ip == nil {
		if result, err := lookup(ipOrHostnameOrURL); err != nil {
			return err
		} else if result != nil {
			ips = append(ips, result...)
		}

		if parsed, err := url.Parse(ipOrHostnameOrURL); err == nil {
			if result, err := lookup(parsed.Hostname()); err != nil {
				return err
			} else if result != nil {
				ips = append(ips, result...)
			}
		}
	} else {
		ips = append(ips, ip)
	}

	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
			return ErrPrivateIPAddressDisallowed(fmt.Errorf("ip %s is in the private, loopback, or unspecified IP range", ip))
		}
	}

	return nil
}

var _ http.RoundTripper = (*NoInternalIPRoundTripper)(nil)

// NoInternalIPRoundTripper is a RoundTripper that disallows internal IP addresses.
type NoInternalIPRoundTripper struct {
	http.RoundTripper
	internalIPExceptions []string
}

func (n NoInternalIPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	rt := http.DefaultTransport
	if n.RoundTripper != nil {
		rt = n.RoundTripper
	}

	for _, exception := range n.internalIPExceptions {
		if IncomingRequestURL(request).String() == exception {
			return rt.RoundTrip(request)
		}
	}

	host, _, _ := net.SplitHostPort(request.Host)
	if err := DisallowIPPrivateAddresses(stringsx.Coalesce(host, request.Host)); err != nil {
		return nil, err
	}

	return rt.RoundTrip(request)
}

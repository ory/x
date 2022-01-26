package httpx

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/ory/x/stringsx"

	"github.com/pkg/errors"
)

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

	for _, disabled := range []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fd47:1ed0:805d:59f0::/64",
		"fc00::/7",
		"::1/128",
	} {
		_, cidr, err := net.ParseCIDR(disabled)
		if err != nil {
			return err
		}

		for _, ip := range ips {
			if cidr.Contains(ip) {
				return fmt.Errorf("ip %s is in the %s range", ip, disabled)
			}
		}
	}

	return nil
}

var _ http.RoundTripper = (*NoInternalIPRoundTripper)(nil)

// NoInternalIPRoundTripper is a RoundTripper that disallows internal IP addresses.
type NoInternalIPRoundTripper struct {
	http.RoundTripper
}

func (n NoInternalIPRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	host, _, _ := net.SplitHostPort(request.Host)
	if err := DisallowIPPrivateAddresses(stringsx.Coalesce(host, request.Host)); err != nil {
		return nil, err
	}

	if n.RoundTripper == nil {
		return http.DefaultTransport.RoundTrip(request)
	}

	return n.RoundTripper.RoundTrip(request)
}

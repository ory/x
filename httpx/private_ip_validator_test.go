// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAssociatedIPAllowed(t *testing.T) {
	for _, disallowed := range []string{
		"localhost",
		"https://localhost/foo?bar=baz#zab",
		"127.0.0.0",
		"127.255.255.255",
		"172.16.0.0",
		"172.31.255.255",
		"192.168.0.0",
		"192.168.255.255",
		"10.0.0.0",
		"0.0.0.0",
		"10.255.255.255",
		"::1",
	} {
		t.Run("case="+disallowed, func(t *testing.T) {
			require.Error(t, DisallowIPPrivateAddresses(disallowed))
		})
	}
}

func TestDisallowLocalIPAddressesWhenSet(t *testing.T) {
	require.NoError(t, DisallowIPPrivateAddresses(""))
	require.Error(t, DisallowIPPrivateAddresses("127.0.0.1"))
	require.ErrorAs(t, DisallowIPPrivateAddresses("127.0.0.1"), new(ErrPrivateIPAddressDisallowed))
}

type noOpRoundTripper struct{}

func (n noOpRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{}, nil
}

var _ http.RoundTripper = new(noOpRoundTripper)

func TestAllowExceptions(t *testing.T) {
	rt := &NoInternalIPRoundTripper{RoundTripper: new(noOpRoundTripper), internalIPExceptions: []string{"http://localhost/asdf"}}

	_, err := rt.RoundTrip(&http.Request{
		Host: "localhost",
		URL:  &url.URL{Scheme: "http", Path: "/asdf", Host: "localhost"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	})
	require.NoError(t, err)

	_, err = rt.RoundTrip(&http.Request{
		Host: "localhost",
		URL:  &url.URL{Scheme: "http", Path: "/not-asdf", Host: "localhost"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	})
	require.Error(t, err)
}

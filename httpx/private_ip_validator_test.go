// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
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

type errRoundTripper struct{}

var fakeErr = errors.New("error")

func (n errRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return nil, fakeErr
}

var _ http.RoundTripper = new(errRoundTripper)

func TestInternalRespectsRoundTripper(t *testing.T) {
	rt := &NoInternalIPRoundTripper{RoundTripper: &errRoundTripper{}, internalIPExceptions: []string{
		"https://127.0.0.1/foo",
	}}

	req, err := http.NewRequest("GET", "https://google.com/foo", nil)
	require.NoError(t, err)
	_, err = rt.RoundTrip(req)
	require.ErrorIs(t, err, fakeErr)

	req, err = http.NewRequest("GET", "https://127.0.0.1/foo", nil)
	require.NoError(t, err)
	_, err = rt.RoundTrip(req)
	require.ErrorIs(t, err, fakeErr)
}

func TestAllowExceptions(t *testing.T) {
	rt := &NoInternalIPRoundTripper{internalIPExceptions: []string{"http://localhost/asdf"}}

	_, err := rt.RoundTrip(&http.Request{
		Host: "localhost",
		URL:  &url.URL{Scheme: "http", Path: "/asdf", Host: "localhost"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	})
	// assert that the error is eiher nil or a dial error.
	if err != nil {
		opErr := new(net.OpError)
		require.ErrorAs(t, err, &opErr)
		require.Equal(t, "dial", opErr.Op)
	}

	_, err = rt.RoundTrip(&http.Request{
		Host: "localhost",
		URL:  &url.URL{Scheme: "http", Path: "/not-asdf", Host: "localhost"},
		Header: http.Header{
			"Host": []string{"localhost"},
		},
	})
	require.Error(t, err)
}

func assertErrorContains(msg string) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, i ...interface{}) bool {
		if !assert.Error(t, err, i...) {
			return false
		}
		return assert.Contains(t, err.Error(), msg)
	}
}

func TestNoInternalDialer(t *testing.T) {
	for _, tt := range []struct {
		name      string
		network   string
		address   string
		assertErr assert.ErrorAssertionFunc
	}{{
		name:      "TCP public is allowed",
		network:   "tcp",
		address:   "www.google.de:443",
		assertErr: assert.NoError,
	}, {
		name:      "TCP private is denied",
		network:   "tcp",
		address:   "localhost:443",
		assertErr: assertErrorContains("is not a public IP address"),
	}, {
		name:      "UDP public is denied",
		network:   "udp",
		address:   "www.google.de:443",
		assertErr: assertErrorContains("not a safe network type"),
	}, {
		name:      "UDP public is denied",
		network:   "udp",
		address:   "www.google.de:443",
		assertErr: assertErrorContains("not a safe network type"),
	}, {
		name:      "UNIX sockets are denied",
		network:   "unix",
		address:   "/etc/passwd",
		assertErr: assertErrorContains("not a safe network type"),
	}} {

		t.Run("case="+tt.name, func(t *testing.T) {
			_, err := NoInternalDialer.Dial(tt.network, tt.address)
			tt.assertErr(t, err)
		})
	}
}

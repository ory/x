package httpx

import (
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
}

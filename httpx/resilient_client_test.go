package httpx

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoPrivateIPs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello, world!"))
	}))
	t.Cleanup(ts.Close)
	c := NewResilientClient(
		ResilientClientWithMaxRetry(1),
		ResilientClientDisallowInternalIPs())

	target, err := url.ParseRequestURI(ts.URL)
	require.NoError(t, err)

	_, port, err := net.SplitHostPort(target.Host)
	require.NoError(t, err)

	for _, host := range []string{
		"127.0.0.1",
		"localhost",
		"192.168.178.5",
	} {
		target.Host = host + ":" + port
		t.Logf("%s", target.String())
		_, err := c.Get(target.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is in the")
	}
}

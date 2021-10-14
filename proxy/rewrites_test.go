package proxy

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestRewrites(t *testing.T) {
	t.Run("suite=HeaderRequest", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
		require.NoError(t, err)
		c := &HostConfig{
			CookieHost:   "example.com",
			OriginalHost: "example.com",
			UpstreamHost: "some-project-1234.oryapis.com",
		}

		require.NoError(t, HeaderRequestRewrite(req, c))
	})
}

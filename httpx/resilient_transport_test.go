package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultResilientRoundTripper(t *testing.T) {
	var c int
	const minTime = time.Second

	var uu = func(u string) string {
		return u
	}

	var ns = func(f func(w http.ResponseWriter, r *http.Request)) func(t *testing.T) *httptest.Server {
		return func(t *testing.T) *httptest.Server {
			return httptest.NewServer(http.HandlerFunc(f))
		}
	}

	for k, tc := range []struct {
		ts               func(t *testing.T) *httptest.Server
		rt               *ResilientRoundTripper
		u                func(u string) string
		expectErr        bool
		expectStatusCode int
	}{
		{
			rt: NewDefaultResilientRoundTripper(minTime, time.Second*2),
			ts: ns(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}),
			u:                uu,
			expectErr:        false,
			expectStatusCode: 204,
		},
		{
			rt: NewDefaultResilientRoundTripper(minTime, time.Second*2),
			ts: ns(func(w http.ResponseWriter, r *http.Request) {
				c++
				if c < 2 {
					w.WriteHeader(500)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}),
			u:                uu,
			expectErr:        false,
			expectStatusCode: http.StatusNoContent,
		},
		{
			rt: NewDefaultResilientRoundTripper(minTime, minTime),
			ts: ns(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(minTime*3 - (minTime * time.Duration(c)))
				c++
				w.WriteHeader(http.StatusNoContent)
			}),
			u:         uu,
			expectErr: true,
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			ts := tc.ts(t)
			defer ts.Close()

			c := http.Client{Timeout: minTime, Transport: tc.rt}
			res, err := c.Get(tc.u(ts.URL))
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "%+v", err)
				assert.Equal(t, tc.expectStatusCode, res.StatusCode)
			}
		})
	}
}

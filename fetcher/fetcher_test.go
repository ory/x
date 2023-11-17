// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/gobuffalo/httptest"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher(t *testing.T) {
	router := httprouter.New()
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		_, _ = w.Write([]byte(`{"foo":"bar"}`))
	})
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	file, err := os.CreateTemp(os.TempDir(), "source.*.json")
	require.NoError(t, err)

	_, err = file.WriteString(`{"foo":"baz"}`)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	rClient := retryablehttp.NewClient()
	rClient.HTTPClient = ts.Client()
	for fc, fetcher := range []*Fetcher{
		NewFetcher(WithClient(rClient)),
		NewFetcher(),
	} {
		for k, tc := range []struct {
			source string
			expect string
		}{
			{
				source: "base64://" + base64.StdEncoding.EncodeToString([]byte(`{"foo":"zab"}`)),
				expect: `{"foo":"zab"}`,
			},
			{
				source: "file://" + file.Name(),
				expect: `{"foo":"baz"}`,
			},
			{
				source: ts.URL,
				expect: `{"foo":"bar"}`,
			},
		} {
			t.Run(fmt.Sprintf("config=%d/case=%d", fc, k), func(t *testing.T) {
				actual, err := fetcher.Fetch(tc.source)
				require.NoError(t, err)
				assert.JSONEq(t, tc.expect, actual.String())
			})
		}
	}

	t.Run("case=returns proper error on unknown scheme", func(t *testing.T) {
		_, err := NewFetcher().Fetch("unknown-scheme://foo")

		assert.ErrorIs(t, err, ErrUnknownScheme)
		assert.Contains(t, err.Error(), "unknown-scheme")
	})

	t.Run("case=FetcherContext cancels the HTTP request", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := NewFetcher().FetchContext(ctx, "https://config.invalid")

		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("case=with-limit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(bytes.Repeat([]byte("test"), 1000))
		}))
		t.Cleanup(srv.Close)

		_, err := NewFetcher(WithMaxHTTPMaxBytes(3999)).Fetch(srv.URL)
		assert.ErrorIs(t, err, bytes.ErrTooLarge)

		_, err = NewFetcher(WithMaxHTTPMaxBytes(4000)).Fetch(srv.URL)
		assert.NoError(t, err)
	})
}

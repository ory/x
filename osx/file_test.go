// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package osx

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/httpx"
)

var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("hello world"))
}

func TestReadFileFromAllSources(t *testing.T) {
	ts := httptest.NewServer(handler)
	defer ts.Close()

	sslTS := httptest.NewTLSServer(handler)
	defer sslTS.Close()

	for k, tc := range []struct {
		opts                []Option
		src                 string
		expectedErr         string
		expectedErrContains string
		expectedBody        string
	}{
		{src: "base64://aGVsbG8gd29ybGQ", expectedBody: "hello world"},
		{src: "base64://aGVsbG8gd29ybGQ=", expectedBody: "hello world", opts: []Option{WithoutResilientBase64Encoding(), WithBase64Encoding(base64.URLEncoding)}},
		{src: "base64://aGVsbG8gd29ybGQ=", expectedErr: "unable to base64 decode the location: illegal base64 data at input byte 15", opts: []Option{WithoutResilientBase64Encoding()}},
		{src: "base64://aGVsbG8gd29ybGQ=", expectedBody: "hello world"},
		{src: "base64://aGVsbG8gd29ybGQ", expectedBody: "hello world"},
		{src: "base64://aGVsbG8gd29ybGQ", expectedErr: "base64 loader disabled", opts: []Option{WithDisabledBase64Loader()}},
		{src: "base64://notbase64", expectedErr: "unable to base64 decode the location: illegal base64 data at input byte 8"},

		{src: "file://stub/text.txt", expectedBody: "hello world"},
		{src: "stub/text.txt", expectedBody: "hello world"},
		{src: "file://stub/text.txt", expectedErr: "file loader disabled", opts: []Option{WithDisabledFileLoader()}},
		{src: "stub/text.txt", expectedErr: "file loader disabled", opts: []Option{WithDisabledFileLoader()}},

		{src: ts.URL, expectedBody: "hello world"},
		{src: sslTS.URL, expectedErrContains: "x509:"},
		{src: sslTS.URL, expectedBody: "hello world", opts: []Option{WithHTTPClient(httpx.NewResilientClient(httpx.ResilientClientWithClient(sslTS.Client())))}},
		{src: sslTS.URL, expectedErr: "http(s) loader disabled", opts: []Option{WithDisabledHTTPLoader()}},

		{src: "file://stub/text.txt", expectedErr: "file loader disabled", opts: []Option{WithDisabledFileLoader()}},

		{src: "lmao://stub/text.txt", expectedErr: "unsupported source `lmao`"},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			body, err := ReadFileFromAllSources(tc.src, tc.opts...)
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
				return
			} else if tc.expectedErrContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestRestrictedReadFile(t *testing.T) {
	ts := httptest.NewServer(handler)
	defer ts.Close()

	sslTS := httptest.NewTLSServer(handler)
	defer sslTS.Close()

	for k, tc := range []struct {
		opts         []Option
		src          string
		expectedErr  string
		expectedBody string
	}{
		{src: "base64://aGVsbG8gd29ybGQ", expectedErr: "base64 loader disabled"},
		{src: "base64://aGVsbG8gd29ybGQ", expectedBody: "hello world", opts: []Option{WithEnabledBase64Loader()}},

		{src: "file://stub/text.txt", expectedErr: "file loader disabled"},
		{src: "file://stub/text.txt", expectedBody: "hello world", opts: []Option{WithEnabledFileLoader()}},

		{src: sslTS.URL, expectedErr: "http(s) loader disabled"},
		{src: ts.URL, expectedBody: "hello world", opts: []Option{WithEnabledHTTPLoader()}},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			body, err := RestrictedReadFile(tc.src, tc.opts...)
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErr, err.Error())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author        Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @Copyright     2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license     Apache-2.0
 */

package healthx

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/herodot"
)

func TestHealth(t *testing.T) {
	const mockHeaderKey = "middleware-header"
	const mockHeaderValue = "test-header-value"
	const mockVersion = "test version"

	// middlware to run an assert function on the requested handler
	testMiddleware := func(t *testing.T, assertFunc func(*testing.T, http.ResponseWriter, *http.Request)) func(next http.Handler) http.Handler {
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				writer.Header().Add(mockHeaderKey, mockHeaderValue)
				assertFunc(t, writer, req)
				h.ServeHTTP(writer, req)
			})
		}
	}

	assertAliveCheck := func(t *testing.T, endpoint string, handler *Handler) *http.Response {
		var healthBody swaggerHealthStatus
		c := http.DefaultClient
		response, err := c.Get(endpoint)
		require.NoError(t, err)
		require.EqualValues(t, http.StatusOK, response.StatusCode)
		require.NoError(t, json.NewDecoder(response.Body).Decode(&healthBody))
		assert.EqualValues(t, "ok", healthBody.Status)
		return response
	}

	assertVersionResponse := func(t *testing.T, endpoint string, handler *Handler) *http.Response {
		var versionBody swaggerVersion
		c := http.DefaultClient
		response, err := c.Get(endpoint)
		require.NoError(t, err)
		require.EqualValues(t, http.StatusOK, response.StatusCode)
		require.NoError(t, json.NewDecoder(response.Body).Decode(&versionBody))
		require.EqualValues(t, mockVersion, versionBody.Version)
		return response
	}

	assertReadyCheckNotAlive := func(t *testing.T, endpoint string, handler *Handler) *http.Response {
		handler.ReadyChecks = map[string]ReadyChecker{
			"test": func(r *http.Request) error {
				return errors.New("not alive")
			},
		}
		c := http.DefaultClient
		response, err := c.Get(endpoint)
		require.NoError(t, err)
		require.EqualValues(t, http.StatusServiceUnavailable, response.StatusCode)
		out, err := ioutil.ReadAll(response.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"errors":{"test":"not alive"}}`, strings.TrimSpace(string(out)))
		return response
	}

	assertReadyCheck := func(t *testing.T, endpoint string, handler *Handler) *http.Response {
		handler.ReadyChecks = map[string]ReadyChecker{
			"test": func(r *http.Request) error {
				return nil
			},
		}
		var healthCheck swaggerHealthStatus
		c := http.DefaultClient
		response, err := c.Get(endpoint)
		require.NoError(t, err)
		require.EqualValues(t, http.StatusOK, response.StatusCode)
		require.NoError(t, json.NewDecoder(response.Body).Decode(&healthCheck))
		require.EqualValues(t, swaggerHealthStatus{Status: "ok"}, healthCheck)
		return response
	}

	testCases := []struct {
		description string
		url         func(mockServerURL string) string
		test        func(t *testing.T, endpoint string, handler *Handler) *http.Response
	}{
		{
			description: "ready check should return status ok",
			url: func(mockServerURL string) string {
				return mockServerURL + ReadyCheckPath
			},
			test: assertReadyCheck,
		},
		{
			description: "ready check should return error",
			url: func(mockServerURL string) string {
				return mockServerURL + ReadyCheckPath
			},
			test: assertReadyCheckNotAlive,
		},
		{
			description: "alive check should return status ok",
			url: func(mockServerURL string) string {
				return mockServerURL + AliveCheckPath
			},
			test: assertAliveCheck,
		},
		{
			description: "version should return",
			url: func(mockServerURL string) string {
				return mockServerURL + VersionPath
			},
			test: assertVersionResponse,
		},
	}

	t.Run("case=without middleware", func(t *testing.T) {
		router := httprouter.New()

		handler := &Handler{
			H:             herodot.NewJSONWriter(nil),
			VersionString: mockVersion,
			ReadyChecks: map[string]ReadyChecker{
				"test": func(r *http.Request) error {
					return nil
				},
			},
		}

		ts := httptest.NewServer(router)
		defer ts.Close()

		handler.SetHealthRoutes(router, true)
		handler.SetVersionRoutes(router)

		for _, tc := range testCases {
			t.Run("case="+tc.description, func(t *testing.T) {
				tc.test(t, tc.url(ts.URL), handler)
			})
		}
	})

	t.Run("case=with middleware", func(t *testing.T) {
		router := httprouter.New()

		var alive error

		handler := &Handler{
			H:             herodot.NewJSONWriter(nil),
			VersionString: mockVersion,
			ReadyChecks: map[string]ReadyChecker{
				"test": func(r *http.Request) error {
					return alive
				},
			},
		}

		ts := httptest.NewServer(router)
		defer ts.Close()

		// set the health handlers with middleware
		handler.SetHealthRoutes(router, true, WithMiddleware(
			testMiddleware(t, func(t *testing.T, rw http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			}),
		))

		handler.SetVersionRoutes(router, WithMiddleware(
			testMiddleware(t, func(t *testing.T, rw http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			}),
		))

		for _, tc := range testCases {
			t.Run("case="+tc.description, func(t *testing.T) {
				response := tc.test(t, tc.url(ts.URL), handler)
				assert.EqualValues(t, mockHeaderValue, response.Header.Get(mockHeaderKey))
			})
		}
	})
}

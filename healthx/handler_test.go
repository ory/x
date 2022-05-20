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

const mockHeaderKey = "middleware-header"
const mockHeaderValue = "test-header-value"

func coreHealthTests(t *testing.T, withMiddleware bool) {
	alive := errors.New("not alive")

	handler := &Handler{
		H:             herodot.NewJSONWriter(nil),
		VersionString: "test version",
		ReadyChecks: map[string]ReadyChecker{
			"test": func(r *http.Request) error {
				return alive
			},
		},
	}

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

	// helper to run the same assertion on various responses that are with the middleware,
	// avoid having to write the same if condition and assertion again
	assertMockHeader := func(t *testing.T, res *http.Response, withMiddleware bool) {
		if withMiddleware == false {
			return
		}
		assert.EqualValues(t, mockHeaderValue, res.Header.Get(mockHeaderKey))
	}

	router := httprouter.New()

	if withMiddleware {
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
	} else {
		handler.SetHealthRoutes(router, true)
		handler.SetVersionRoutes(router)
	}

	ts := httptest.NewServer(router)

	c := http.DefaultClient

	var healthBody swaggerHealthStatus
	response, err := c.Get(ts.URL + AliveCheckPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, response.StatusCode)
	require.NoError(t, json.NewDecoder(response.Body).Decode(&healthBody))
	assert.EqualValues(t, "ok", healthBody.Status)
	assertMockHeader(t, response, withMiddleware)

	var versionBody swaggerVersion
	response, err = c.Get(ts.URL + VersionPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, response.StatusCode)
	require.NoError(t, json.NewDecoder(response.Body).Decode(&versionBody))
	require.EqualValues(t, versionBody.Version, handler.VersionString)
	assertMockHeader(t, response, withMiddleware)

	response, err = c.Get(ts.URL + ReadyCheckPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusServiceUnavailable, response.StatusCode)
	out, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)
	assert.EqualValues(t, "ok", healthBody.Status)
	assert.Equal(t, `{"errors":{"test":"not alive"}}`, strings.TrimSpace(string(out)))
	assertMockHeader(t, response, withMiddleware)

	alive = nil
	response, err = c.Get(ts.URL + ReadyCheckPath)
	require.NoError(t, err)
	require.EqualValues(t, http.StatusOK, response.StatusCode)
	require.NoError(t, json.NewDecoder(response.Body).Decode(&versionBody))
	require.EqualValues(t, versionBody.Version, handler.VersionString)
	assertMockHeader(t, response, withMiddleware)
}

func TestHealth(t *testing.T) {
	t.Run("case:without middleware", func(t *testing.T) {
		coreHealthTests(t, false)
	})

	t.Run("case:with middleware", func(t *testing.T) {
		coreHealthTests(t, true)
	})
}

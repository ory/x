/*
 * Copyright © 2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
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
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2017-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */
package pagination

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Run("case=normal", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo?limit=10&offset=10")
		limit, offset := Parse(&http.Request{URL: u}, 0, 0, 10)
		assert.EqualValues(t, limit, 10)
		assert.EqualValues(t, offset, 10)
	})

	t.Run("case=defaults", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo")
		limit, offset := Parse(&http.Request{URL: u}, 5, 5, 10)
		assert.EqualValues(t, limit, 5)
		assert.EqualValues(t, offset, 5)
	})

	t.Run("case=defaults_and_limits", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo")
		limit, offset := Parse(&http.Request{URL: u}, 5, 5, 2)
		assert.EqualValues(t, limit, 2)
		assert.EqualValues(t, offset, 5)
	})

	t.Run("case=limits", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo?limit=10&offset=10")
		limit, offset := Parse(&http.Request{URL: u}, 0, 0, 5)
		assert.EqualValues(t, limit, 5)
		assert.EqualValues(t, offset, 10)
	})

	t.Run("case=negatives", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo?limit=-1&offset=-1")
		limit, offset := Parse(&http.Request{URL: u}, 0, 0, 5)
		assert.EqualValues(t, limit, 0)
		assert.EqualValues(t, offset, 0)
	})

	t.Run("case=default_negatives", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo")
		limit, offset := Parse(&http.Request{URL: u}, -1, -1, 5)
		assert.EqualValues(t, limit, 0)
		assert.EqualValues(t, offset, 0)
	})

	t.Run("case=invalid_defaults", func(t *testing.T) {
		u, _ := url.Parse("http://localhost/foo?offset=a&limit=b")
		limit, offset := Parse(&http.Request{URL: u}, 10, 10, 15)
		assert.EqualValues(t, limit, 10)
		assert.EqualValues(t, offset, 10)
	})
}

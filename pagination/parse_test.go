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
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		d   string
		url string
		dl  int
		do  int
		ml  int
		el  int
		eo  int
	}{
		{"normal", "http://localhost/foo?limit=10&offset=10", 0, 0, 120, 10, 10},
		{"defaults", "http://localhost/foo", 5, 5, 10, 5, 5},
		{"defaults_and_limits", "http://localhost/foo", 5, 5, 2, 2, 5},
		{"limits", "http://localhost/foo?limit=10&offset=10", 0, 0, 5, 5, 10},
		{"negatives", "http://localhost/foo?limit=-1&offset=-1", 0, 0, 5, 0, 0},
		{"default_negatives", "http://localhost/foo", -1, -1, 5, 0, 0},
		{"invalid_defaults", "http://localhost/foo?limit=a&offset=b", 10, 10, 15, 10, 10},
	} {
		t.Run(fmt.Sprintf("case=%s", tc.d), func(t *testing.T) {
			u, _ := url.Parse(tc.url)
			limit, offset := Parse(&http.Request{URL: u}, tc.dl, tc.do, tc.ml)
			assert.EqualValues(t, limit, tc.el)
			assert.EqualValues(t, offset, tc.eo)
		})
	}
}

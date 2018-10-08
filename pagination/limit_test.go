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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIndex(t *testing.T) {
	for k, c := range []struct {
		s      []string
		offset int
		limit  int
		e      []string
	}{
		{
			s:      []string{"a", "b", "c"},
			offset: 0,
			limit:  100,
			e:      []string{"a", "b", "c"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 0,
			limit:  2,
			e:      []string{"a", "b"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 1,
			limit:  10,
			e:      []string{"b", "c"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 1,
			limit:  2,
			e:      []string{"b", "c"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 2,
			limit:  2,
			e:      []string{"c"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 3,
			limit:  10,
			e:      []string{},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 2,
			limit:  10,
			e:      []string{"c"},
		},
		{
			s:      []string{"a", "b", "c"},
			offset: 1,
			limit:  10,
			e:      []string{"b", "c"},
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			start, end := Index(c.limit, c.offset, len(c.s))
			assert.EqualValues(t, c.e, c.s[start:end])
		})
	}
}

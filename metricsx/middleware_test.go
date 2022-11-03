// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

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
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package metricsx

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnonymizePath(t *testing.T) {
	m := &Service{
		o: &Options{WhitelistedPaths: []string{"/keys"}},
	}

	assert.Equal(t, "/keys/837b4168b57215f2ba0d4e64e57a653d6a6ecd6065e78598283209467d172373", m.anonymizePath("/keys/1234", "somesupersaltysalt"))
	assert.Equal(t, "/keys", m.anonymizePath("/keys", "somesupersaltysalt"))
}

func TestAnonymizeQuery(t *testing.T) {
	m := &Service{}

	assert.EqualValues(t, "foo=2ec879270efe890972d975251e9d454f4af49df1f07b4317fd5b6ae90de4c774&foo=1864a573566eba1b9ddab79d8f4bab5a39c938918a21b80a64ae1c9c12fa9aa2&foo2=186084f6bd8e222bedade9439d6ae69ed274b954eeebe9b54fd5f47e54dd7675&foo2=1ee7158281cc3b5a27de4c337e07987e8677f5f687a4671ca369b79c653d379d", m.anonymizeQuery(url.Values{
		"foo":  []string{"bar", "baz"},
		"foo2": []string{"bar2", "baz2"},
	}, "somesupersaltysalt"))
	assert.EqualValues(t, "", m.anonymizeQuery(url.Values{
		"foo": []string{},
	}, "somesupersaltysalt"))
	assert.EqualValues(t, "foo=", m.anonymizeQuery(url.Values{
		"foo": []string{""},
	}, "somesupersaltysalt"))
	assert.EqualValues(t, "", m.anonymizeQuery(url.Values{}, "somesupersaltysalt"))
}

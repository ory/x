// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package assertx

import "testing"

func TestEqualAsJSONExcept(t *testing.T) {
	a := map[string]interface{}{"foo": "bar", "baz": "bar", "bar": "baz"}
	b := map[string]interface{}{"foo": "bar", "baz": "bar", "bar": "not-baz"}

	EqualAsJSONExcept(t, a, b, []string{"bar"})
}

// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package sqlxx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandablesHas(t *testing.T) {
	var e = Expandables{"foo", "bar"}
	assert.True(t, e.Has("foo"))
	assert.True(t, e.Has("bar"))
	assert.False(t, e.Has("baz"))
}

func TestExpandablesToEager(t *testing.T) {
	assert.Equal(t, []string{"foo", "bar"}, Expandables{"foo", "bar"}.ToEager())
}

func TestExpandablesSort(t *testing.T) {
	var e = Expandables{
		"third_1.third_2.third_3",
		"first_1",
		"first_1.first_2.first_3_2",
		"first_1.first_2.first_3_1",
		"first_1.first_2",
		"third_1",
		"second_1",
		"third_1.third_2",
	}
	e.Sort()
	assert.Equal(t, Expandables{
		"first_1",
		"second_1",
		"third_1",
		"first_1.first_2",
		"third_1.third_2",
		"first_1.first_2.first_3_1",
		"first_1.first_2.first_3_2",
		"third_1.third_2.third_3",
	}, e)
}

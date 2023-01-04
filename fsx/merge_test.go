// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package fsx

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestMergeFS(t *testing.T) {
	a := fstest.MapFS{
		"a":     &fstest.MapFile{},
		"dir/c": &fstest.MapFile{},
	}
	b := fstest.MapFS{
		"b":     &fstest.MapFile{},
		"dir/d": &fstest.MapFile{},
	}
	m := Merge(a, b)

	assert.NoError(t, fstest.TestFS(m, "a", "b", "dir", "dir/c", "dir/d"))
}

// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package fsx

import (
	"testing"
	"testing/fstest"

	"github.com/laher/mergefs"
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

	x := fstest.MapFS{
		"x":     &fstest.MapFile{},
		"dir/y": &fstest.MapFile{},
	}

	m2 := Merge(m, x)
	assert.NoError(t, fstest.TestFS(m2, "a", "b", "dir", "dir/c", "dir/d", "dir/y", "x"))

	m2 = mergefs.Merge(mergefs.Merge(a, b), x)
	assert.NoError(t, fstest.TestFS(m2, "a", "b", "dir", "dir/c", "dir/d", "dir/y", "x"))
}

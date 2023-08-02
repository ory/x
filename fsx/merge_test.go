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
	x := fstest.MapFS{
		"x":     &fstest.MapFile{},
		"dir/y": &fstest.MapFile{},
	}

	assert.NoError(t, fstest.TestFS(
		Merge(a, b),
		"a",
		"b",
		"dir",
		"dir/c",
		"dir/d",
	))

	assert.NoError(t, fstest.TestFS(
		Merge(a, b, x),
		"a",
		"b",
		"dir",
		"dir/c",
		"dir/d",
		"dir/y",
		"x",
	))

	assert.Error(t, fstest.TestFS(
		mergefs.Merge(a, b),
		"a",
		"b",
		"dir",
		"dir/c",
		"dir/d",
	))

	assert.Error(t, fstest.TestFS(
		mergefs.Merge(mergefs.Merge(a, b), x),
		"a",
		"b",
		"dir",
		"dir/c",
		"dir/d",
		"dir/y",
		"x",
	))
}

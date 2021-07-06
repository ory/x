package fsx

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

//go:embed merge.go
var prodFS embed.FS

//go:embed merge_test.go
var testFS embed.FS

func TestMergeFS(t *testing.T) {
	mergedFS := Merge(prodFS, testFS)

	_, err := mergedFS.Open("merge.go")
	assert.NoError(t, err)
	_, err = mergedFS.Open("merge_test.go")
	assert.NoError(t, err)
	_, err = mergedFS.Open("unknown file")
	assert.True(t, errors.Is(err, fs.ErrNotExist))
}

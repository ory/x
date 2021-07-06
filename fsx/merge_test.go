package fsx

import (
	"embed"
	"testing"

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
}

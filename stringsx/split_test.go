package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitNonEmpty(t *testing.T) {
	// assert.Len(t, strings.Split("", " "), 1)
	assert.Len(t, Splitx("", " "), 0)
}

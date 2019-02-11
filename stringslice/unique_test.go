package stringslice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	assert.EqualValues(t, []string{"foo", "bar", "baz"}, Unique([]string{"foo", "foo", "bar", "baz", "bar"}))
}

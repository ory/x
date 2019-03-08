package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToLowerInitial(t *testing.T) {
	assert.Equal(t, "", ToLowerInitial(""))
	assert.Equal(t, "a", ToLowerInitial("a"))
	assert.Equal(t, "a", ToLowerInitial("A"))
	assert.Equal(t, "ab", ToLowerInitial("Ab"))
	assert.Equal(t, "aA", ToLowerInitial("AA"))
}

func TestToUpperInitial(t *testing.T) {
	assert.Equal(t, "", ToUpperInitial(""))
	assert.Equal(t, "A", ToUpperInitial("a"))
	assert.Equal(t, "A", ToUpperInitial("A"))
	assert.Equal(t, "AB", ToUpperInitial("aB"))
	assert.Equal(t, "Ab", ToUpperInitial("ab"))
}

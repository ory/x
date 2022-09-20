package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateString(t *testing.T) {
	s := "HelloWorld"
	res := TruncateByLength(s, 7)
	assert.Equal(t, "HelloWo", res)
}

func TestTruncateString_LongerThanString(t *testing.T) {
	s := "HelloWorld"
	res := TruncateByLength(s, 15)
	assert.Equal(t, s, res)
}

func TestTruncateString_InvalidLength(t *testing.T) {
	s := "HelloWorld"
	res := TruncateByLength(s, -1)
	assert.Equal(t, s, res)
}

package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPointer(t *testing.T) {
	s := "TestString"
	assert.Equal(t, &s, GetPointer(s))
}

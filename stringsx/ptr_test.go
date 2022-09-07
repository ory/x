package stringsx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPointer(t *testing.T) {
	s := "TestString"
	assert.Equal(t, &s, GetPointer(s))
}

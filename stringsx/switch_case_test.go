package stringsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisteredCases(t *testing.T) {
	t.Run("case=adds values", func(t *testing.T) {
		v1, v2 := "value 1", "value 2"

		cs := RegisteredCases{}
		cs.AddCase(v1)
		cs.AddCase(v2)

		assert.Equal(t, RegisteredCases{v1, v2}, cs)
	})

	t.Run("case=returns value on add", func(t *testing.T) {
		v1, v2 := "value 1", "value 2"

		cs := RegisteredCases{}
		assert.Equal(t, v1, cs.AddCase(v1))
		assert.Equal(t, v2, cs.AddCase(v2))
	})

	t.Run("case=switch integration", func(t *testing.T) {
		cases := RegisteredCases{}

		switch "foo" {
		case cases.AddCase("bar"):
		case cases.AddCase("baz"):
		}

		assert.Equal(t, RegisteredCases{"bar", "baz"}, cases)
	})
}

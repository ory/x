package stringsx

import (
	"errors"
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

	t.Run("case=converts to correct error", func(t *testing.T) {
		c1, c2, actual := "case 1", "case 2", "actual"

		cs := RegisteredCases{}
		cs.AddCase(c1)
		cs.AddCase(c2)

		err := cs.ToUnknownCaseErr(actual)

		assert.True(t, errors.Is(err, ErrUnknownCase))
		assert.Equal(t, errUnknownCase{
			cases:  cs,
			actual: actual,
		}, err)
	})

	t.Run("case=switch integration", func(t *testing.T) {
		cases := RegisteredCases{}
		var err error

		switch f := "foo"; f {
		case cases.AddCase("bar"):
		case cases.AddCase("baz"):
		default:
			err = cases.ToUnknownCaseErr(f)
		}

		assert.Equal(t, RegisteredCases{"bar", "baz"}, cases)
		assert.Equal(t, errUnknownCase{
			cases:  cases,
			actual: "foo",
		}, err)
	})
}

package jsonschemax

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONPointerToDotNotation(t *testing.T) {
	for _, tc := range [][]string{
		{"#/foo/bar/baz", "foo.bar.baz"},
		{"#/baz", "baz"},
	} {
		require.Equal(t, tc[1], JSONPointerToDotNotation(tc[0]))
	}
}

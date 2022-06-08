// Package testingx contains helper functions and extensions used when writing tests in Ory.
package testingx

import (
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

// ReadAll reads all bytes from the reader and returns them as a byte slice.
func ReadAll(t testing.TB, r io.Reader) []byte {
	body, err := io.ReadAll(r)
	require.NoError(t, err)
	return body
}

// ReadAllString reads all bytes from the reader and returns them as a string.
func ReadAllString(t testing.TB, r io.Reader) string {
	return string(ReadAll(t, r))
}

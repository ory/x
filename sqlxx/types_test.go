package sqlxx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullTime(t *testing.T) {
	out, err := json.Marshal(NullTime{})
	require.NoError(t, err)
	assert.EqualValues(t, "null", string(out))
}

func TestNullString_UnmarshalJSON(t *testing.T) {
	data := []byte(`"hello"`)
	var ns NullString
	require.NoError(t, json.Unmarshal(data, &ns))
	assert.EqualValues(t, "hello", ns)
}

func TestStringSlicePipeDelimiter(t *testing.T) {
	expected := StringSlicePipeDelimiter([]string{"foo", "bar|baz", "zab"})
	encoded, err := expected.Value()
	require.NoError(t, err)
	var actual StringSlicePipeDelimiter
	require.NoError(t, actual.Scan(encoded))
	assert.Equal(t, expected, actual)
}

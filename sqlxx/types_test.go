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

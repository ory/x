package sjsonx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetBytes(t *testing.T) {
	out, err := SetBytes([]byte(`{"a":1,"b":2,"c":3}`), map[string]interface{}{"d.e": "6", "d.f": "7"})
	require.NoError(t, err)
	assert.EqualValues(t, string(out), `{"a":1,"b":2,"c":3,"d":{"e":"6","f":"7"}}`)
}

func TestSet(t *testing.T) {
	out, err := Set(`{"a":1,"b":2,"c":3}`, map[string]interface{}{"d.e": "6", "d.f": "7"})
	require.NoError(t, err)
	assert.EqualValues(t, out, `{"a":1,"b":2,"c":3,"d":{"e":"6","f":"7"}}`)
}

package sjsonx

import (
	"encoding/json"
	"github.com/ory/x/assertx"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetBytes(t *testing.T) {
	out, err := SetBytes([]byte(`{"a":1,"b":2,"c":3}`), map[string]interface{}{"d.e": "6", "d.f": "7"})
	require.NoError(t, err)
	assertx.EqualAsJSON(t, json.RawMessage(`{"a":1,"b":2,"c":3,"d":{"e":"6","f":"7"}}`), json.RawMessage(out))
}

func TestSet(t *testing.T) {
	out, err := Set(`{"a":1,"b":2,"c":3}`, map[string]interface{}{"d.e": "6", "d.f": "7"})
	require.NoError(t, err)
	assertx.EqualAsJSON(t, json.RawMessage(`{"a":1,"b":2,"c":3,"d":{"e":"6","f":"7"}}`), json.RawMessage(out))
}

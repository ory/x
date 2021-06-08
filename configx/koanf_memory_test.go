package configx

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ory/x/assertx"
)

func TestKoanfMemory(t *testing.T) {
	doc := []byte(`{
  "foo": {
    "bar": "baz"
  }
}`)
	kf := NewKoanfMemory(context.Background(), doc)

	actual, err := kf.Read()
	require.NoError(t, err)
	assertx.EqualAsJSON(t, json.RawMessage(doc), actual)
}

package sqlxx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type data struct {
	Foo string `json:"foo"`
}

func TestJSONScan(t *testing.T) {
	var d data

	// []byte input
	require.NoError(t, JSONScan(&d, []byte(`{"foo": "some value"}`)))
	assert.Equal(t, "some value", d.Foo)

	// string input
	require.NoError(t, JSONScan(&d, `{"foo": "other value"}`))
	assert.Equal(t, "other value", d.Foo)
}

func TestJSONValue(t *testing.T) {
	d := data{
		Foo: "bar",
	}

	json, err := JSONValue(&d)
	require.NoError(t, err)
	assert.Equal(t, `{"foo":"bar"}`, strings.Join(strings.Fields(json.(string)), ""))
}

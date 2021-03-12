package assertx

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/tidwall/sjson"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func PrettifyJSONPayload(t *testing.T, payload interface{}) string {
	o, err := json.MarshalIndent(payload, "", "  ")
	require.NoError(t, err)
	return string(o)
}

func EqualAsJSON(t *testing.T, expected, actual interface{}, args ...interface{}) {
	var eb, ab bytes.Buffer
	if len(args) == 0 {
		args = []interface{}{PrettifyJSONPayload(t, actual)}
	}

	require.NoError(t, json.NewEncoder(&eb).Encode(expected), args...)
	require.NoError(t, json.NewEncoder(&ab).Encode(actual), args...)
	assert.JSONEq(t, eb.String(), ab.String(), args...)
}

func EqualAsJSONExcept(t *testing.T, expected, actual interface{}, except []string, args ...interface{}) {
	var eb, ab bytes.Buffer
	if len(args) == 0 {
		args = []interface{}{PrettifyJSONPayload(t, actual)}
	}

	require.NoError(t, json.NewEncoder(&eb).Encode(expected), args...)
	require.NoError(t, json.NewEncoder(&ab).Encode(actual), args...)

	var err error
	ebs, abs := eb.String(), ab.String()
	for _, k := range except {
		ebs, err = sjson.Delete(ebs, k)
		require.NoError(t, err)

		abs, err = sjson.Delete(abs, k)
		require.NoError(t, err)
	}

	assert.JSONEq(t, ebs, abs, args...)
}

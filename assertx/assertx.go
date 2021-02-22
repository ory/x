package assertx

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/tidwall/sjson"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func EqualAsJSON(t *testing.T, expected, actual interface{}, args ...interface{}) {
	var eb, ab bytes.Buffer
	require.NoError(t, json.NewEncoder(&eb).Encode(expected))
	require.NoError(t, json.NewEncoder(&ab).Encode(actual))
	assert.JSONEq(t, eb.String(), ab.String(), args...)
}

func EqualAsJSONExcept(t *testing.T, expected, actual interface{}, except []string, args ...interface{}) {
	var eb, ab bytes.Buffer
	require.NoError(t, json.NewEncoder(&eb).Encode(expected))
	require.NoError(t, json.NewEncoder(&ab).Encode(actual))

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

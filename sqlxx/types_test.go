package sqlxx

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullTime(t *testing.T) {
	out, err := json.Marshal(NullTime{})
	require.NoError(t, err)
	assert.EqualValues(t, "null", string(out))
}

func TestDuration(t *testing.T) {
	out, err := json.Marshal(Duration(time.Second))
	require.NoError(t, err)
	assert.EqualValues(t, `"1s"`, string(out))
}

func TestNullString_UnmarshalJSON(t *testing.T) {
	data := []byte(`"hello"`)
	var ns NullString
	require.NoError(t, json.Unmarshal(data, &ns))
	assert.EqualValues(t, "hello", ns)
}

func TestNullBoolMarshalJSON(t *testing.T) {
	type outer struct {
		Bool *NullBool `json:"null_bool,omitempty"`
	}

	for k, tc := range []struct {
		in       *outer
		expected string
	}{
		{in: &outer{&NullBool{Valid: false, Bool: true}}, expected: "{\"null_bool\":null}"},
		{in: &outer{&NullBool{Valid: true, Bool: true}}, expected: "{\"null_bool\":true}"},
		{in: &outer{&NullBool{Valid: true, Bool: false}}, expected: "{\"null_bool\":false}"},
		{in: &outer{}, expected: "{}"},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			out, err := json.Marshal(tc.in)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expected, string(out))
		})
	}
}

func TestNullBoolUnMarshalJSONNoPointer(t *testing.T) {
	type outer struct {
		Bool NullBool `json:"null_bool,omitempty"`
	}

	for k, tc := range []struct {
		expected outer
		in       string
	}{
		{expected: outer{}, in: "{}"},
		{expected: outer{NullBool{Valid: true, Bool: true}}, in: "{\"null_bool\":true}"},
		{expected: outer{NullBool{Valid: true, Bool: false}}, in: "{\"null_bool\":false}"},
		{expected: outer{NullBool{}}, in: "{\"null_bool\":null}"},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			var actual outer
			err := json.Unmarshal([]byte(tc.in), &actual)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expected, actual)
		})
	}
}

func TestNullBoolUnMarshalJSON(t *testing.T) {
	type outer struct {
		Bool *NullBool `json:"null_bool,omitempty"`
	}

	for k, tc := range []struct {
		expected outer
		in       string
	}{
		{expected: outer{}, in: "{}"},
		{expected: outer{&NullBool{Valid: true, Bool: true}}, in: "{\"null_bool\":true}"},
		{expected: outer{&NullBool{Valid: true, Bool: false}}, in: "{\"null_bool\":false}"},
		{expected: outer{}, in: "{\"null_bool\":null}"},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			var actual outer
			err := json.Unmarshal([]byte(tc.in), &actual)
			require.NoError(t, err)
			assert.EqualValues(t, tc.expected, actual)
		})
	}
}

func TestStringSlicePipeDelimiter(t *testing.T) {
	expected := StringSlicePipeDelimiter([]string{"foo", "bar|baz", "zab"})
	encoded, err := expected.Value()
	require.NoError(t, err)
	var actual StringSlicePipeDelimiter
	require.NoError(t, actual.Scan(encoded))
	assert.Equal(t, expected, actual)
}

func TestJSONArrayRawMessage(t *testing.T) {
	expected, err := JSONArrayRawMessage("").Value()
	require.NoError(t, err)
	assert.EqualValues(t, "[]", fmt.Sprintf("%s", expected))

	expected, err = JSONArrayRawMessage("null").Value()
	require.NoError(t, err)
	assert.EqualValues(t, "[]", fmt.Sprintf("%s", expected))

	_, err = JSONArrayRawMessage("{}").Value()
	require.Error(t, err)

	expected, err = JSONArrayRawMessage(`["foo","bar"]`).Value()
	require.NoError(t, err)
	assert.EqualValues(t, `["foo","bar"]`, fmt.Sprintf("%s", expected))

	var v JSONArrayRawMessage
	require.Error(t, v.Scan("{}"))

	require.NoError(t, v.Scan(""))
	assert.EqualValues(t, "[]", string(v))

	require.NoError(t, v.Scan("null"))
	assert.EqualValues(t, "[]", string(v))

	require.NoError(t, v.Scan(`["foo","bar"]`))
	assert.EqualValues(t, `["foo","bar"]`, string(v))
}

func TestStringSliceJSONFormat(t *testing.T) {
	expected, err := StringSliceJSONFormat{}.Value()
	require.NoError(t, err)
	assert.EqualValues(t, "[]", fmt.Sprintf("%s", expected))

	expected, err = StringSliceJSONFormat{"foo", "bar"}.Value()
	require.NoError(t, err)
	assert.EqualValues(t, `["foo","bar"]`, fmt.Sprintf("%s", expected))

	var v StringSliceJSONFormat
	require.Error(t, v.Scan("{}"))

	require.NoError(t, v.Scan(""))
	assert.Empty(t, v)

	require.NoError(t, v.Scan("null"))
	assert.Empty(t, v)

	require.NoError(t, v.Scan(`["foo","bar"]`))
	assert.EqualValues(t, StringSliceJSONFormat{"foo", "bar"}, v)
}

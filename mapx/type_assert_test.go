package mapx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetString(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": 1234}
	v, err := GetString(m, "foo")
	require.NoError(t, err)
	assert.EqualValues(t, "bar", v)
	v, err = GetString(m, "bar")
	require.Error(t, err)
	v, err = GetString(m, "baz")
	require.Error(t, err)
}

func TestGetStringSlice(t *testing.T) {
	m := map[interface{}]interface{}{"foo": []string{"foo", "bar"}, "baz": "bar"}
	v, err := GetStringSlice(m, "foo")
	require.NoError(t, err)
	assert.EqualValues(t, []string{"foo", "bar"}, v)
	v, err = GetStringSlice(m, "bar")
	require.Error(t, err)
	v, err = GetStringSlice(m, "baz")
	require.Error(t, err)
}

func TestGetStringSliceDefault(t *testing.T) {
	m := map[interface{}]interface{}{"foo": []string{"foo", "bar"}, "baz": "bar"}
	assert.EqualValues(t, []string{"foo", "bar"}, GetStringSliceDefault(m, "foo", []string{"default"}))
	assert.EqualValues(t, []string{"default"}, GetStringSliceDefault(m, "baz", []string{"default"}))
	assert.EqualValues(t, []string{"default"}, GetStringSliceDefault(m, "bar", []string{"default"}))
}

func TestGetStringDefault(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": 1234}
	assert.EqualValues(t, "bar", GetStringDefault(m, "foo", "default"))
	assert.EqualValues(t, "default", GetStringDefault(m, "baz", "default"))
	assert.EqualValues(t, "default", GetStringDefault(m, "bar", "default"))
}

func TestGetFloat32(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": float32(1234)}
	v, err := GetFloat32(m, "baz")
	require.NoError(t, err)
	assert.EqualValues(t, float32(1234), v)
	v, err = GetFloat32(m, "foo")
	require.Error(t, err)
	v, err = GetFloat32(m, "bar")
	require.Error(t, err)
}

func TestGetFloat64(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": float64(1234)}
	v, err := GetFloat64(m, "baz")
	require.NoError(t, err)
	assert.EqualValues(t, float64(1234), v)
	v, err = GetFloat64(m, "foo")
	require.Error(t, err)
	v, err = GetFloat64(m, "bar")
	require.Error(t, err)
}

func TestGetInt64(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": int64(1234)}
	v, err := GetInt64(m, "baz")
	require.NoError(t, err)
	assert.EqualValues(t, int64(1234), v)
	v, err = GetInt64(m, "foo")
	require.Error(t, err)
	v, err = GetInt64(m, "bar")
	require.Error(t, err)
}

func TestGetInt32(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": int32(1234), "baz2": int(1234)}
	v, err := GetInt32(m, "baz")
	require.NoError(t, err)
	assert.EqualValues(t, int32(1234), v)
	v, err = GetInt32(m, "baz2")
	require.NoError(t, err)
	assert.EqualValues(t, int32(1234), v)
	v, err = GetInt32(m, "foo")
	require.Error(t, err)
	v, err = GetInt32(m, "bar")
	require.Error(t, err)
}

func TestGetInt(t *testing.T) {
	m := map[interface{}]interface{}{"foo": "bar", "baz": 1234, "baz2": int32(1234)}
	v, err := GetInt32(m, "baz")
	require.NoError(t, err)
	assert.EqualValues(t, int32(1234), v)
	v, err = GetInt32(m, "foo")
	require.Error(t, err)
	v, err = GetInt32(m, "bar")
	require.Error(t, err)
}

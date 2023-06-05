// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"testing"

	"github.com/mohae/deepcopy"
	"github.com/stretchr/testify/require"
)

type TestType struct {
	Field1 string
	Field2 []string
	Field3 struct {
		Field1 bool
		Field2 []int
	}
	FieldNull *struct {
		Field1 any
	}
}

func TestApplyJSONPatch(t *testing.T) {
	object := TestType{
		Field1: "foo",
		Field2: []string{
			"foo",
			"bar",
			"baz",
			"kaz",
		},
		Field3: struct {
			Field1 bool
			Field2 []int
		}{
			Field1: true,
			Field2: []int{
				1,
				2,
				3,
			},
		},
	}
	t.Run("case=empty patch", func(t *testing.T) {
		rawPatch := []byte(`[]`)
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, object, obj)
	})
	t.Run("case=field replace", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "replace", "path": "/Field1", "value": "boo"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field1 = "boo"
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=array replace", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "replace", "path": "/Field2/0", "value": "boo"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field2[0] = "boo"
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=array append", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "add", "path": "/Field2/-", "value": "boo"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field2 = append(expected.Field2, "boo")
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=array remove", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "remove", "path": "/Field2/0"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field2 = expected.Field2[1:]
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=nested field replace", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "replace", "path": "/Field3/Field1", "value": false}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field3.Field1 = false
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=nested array append", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "add", "path": "/Field3/Field2/-", "value": 4}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field3.Field2 = append(expected.Field3.Field2, 4)
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=nested array remove", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "remove", "path": "/Field3/Field2/2"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field3.Field2 = expected.Field3.Field2[:2]
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj))
		require.Equal(t, expected, obj)
	})
	t.Run("case=patch denied path", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "replace", "path": "/Field1", "value": "bar"}]`)
		obj := deepcopy.Copy(object).(TestType)
		require.Error(t, ApplyJSONPatch(rawPatch, &obj, "/Field1"))
		require.Equal(t, object, obj)
	})
	t.Run("case=patch allowed path", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "add", "path": "/Field2/-", "value": "bar"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.Field2 = append(expected.Field2, "bar")
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj, "/Field1"))
		require.Equal(t, expected, obj)
	})
	t.Run("case=patch object field when object null", func(t *testing.T) {
		rawPatch := []byte(`[{"op": "add", "path": "/FieldNull/Field1", "value": "bar"}]`)
		expected := deepcopy.Copy(object).(TestType)
		expected.FieldNull = &struct {
			Field1 any
		}{}
		expected.FieldNull.Field1 = "bar"
		obj := deepcopy.Copy(object).(TestType)
		require.NoError(t, ApplyJSONPatch(rawPatch, &obj, "/Field1"))
		require.Equal(t, expected, obj)
	})

	t.Run("suite=invalid patches", func(t *testing.T) {
		cases := []struct {
			name  string
			patch []byte
		}{{
			name:  "test",
			patch: []byte(`[{"op": "test", "path": "/"}]`),
		}, {
			name:  "add",
			patch: []byte(`[{"op": "add", "path": "/"}]`),
		}, {
			name:  "remove",
			patch: []byte(`[{"op": "add", "path": "/"}]`),
		}, {
			name:  "replace",
			patch: []byte(`[{"op": "add", "path": "/"}]`),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(_ *testing.T) {
				obj := &TestType{}
				ApplyJSONPatch(tc.patch, &obj)
			})
		}
	})

}

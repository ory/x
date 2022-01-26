package snapshotx

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tidwall/gjson"

	"github.com/ory/x/stringslice"

	"github.com/bradleyjkemp/cupaloy/v2"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
)

func SnapshotTExcept(t *testing.T, actual interface{}, except []string) {
	compare, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err, "%+v", actual)
	for _, e := range except {
		compare, err = sjson.DeleteBytes(compare, e)
		require.NoError(t, err, "%s", e)
	}

	cupaloy.New(
		cupaloy.CreateNewAutomatically(true),
		cupaloy.FailOnUpdate(true),
		cupaloy.SnapshotFileExtension(".json"),
	).SnapshotT(t, compare)
}

func deleteMatches(t *testing.T, key string, result gjson.Result, matches []string, parents []string, content []byte) []byte {
	path := parents
	if key != "" {
		path = append(parents, key)
	}

	if result.IsObject() {
		result.ForEach(func(key, value gjson.Result) bool {
			content = deleteMatches(t, key.String(), value, matches, path, content)
			return true
		})
	} else if result.IsArray() {
		var i int
		result.ForEach(func(_, value gjson.Result) bool {
			content = deleteMatches(t, fmt.Sprintf("%d", i), value, matches, path, content)
			i++
			return true
		})
	}

	if stringslice.Has(matches, key) {
		content, err := sjson.DeleteBytes(content, strings.Join(path, "."))
		require.NoError(t, err)
		return content
	}

	return content
}

// SnapshotTExceptMatchingKeys works like SnapshotTExcept but deletes keys that match the given matches recursively.
//
// So instead of having deeply nested keys like `foo.bar.baz.0.key_to_delete` you can have `key_to_delete` and
// all occurences of `key_to_delete` will be removed.
func SnapshotTExceptMatchingKeys(t *testing.T, actual interface{}, matches []string) {
	compare, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err, "%+v", actual)

	parsed := gjson.ParseBytes(compare)
	require.True(t, parsed.IsObject() || parsed.IsArray())
	compare = deleteMatches(t, "", parsed, matches, []string{}, compare)

	cupaloy.New(
		cupaloy.CreateNewAutomatically(true),
		cupaloy.FailOnUpdate(true),
		cupaloy.SnapshotFileExtension(".json"),
	).SnapshotT(t, compare)
}

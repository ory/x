package jsonnetsecure

import (
	"fmt"
	"testing"

	"github.com/google/go-jsonnet"

	"github.com/ory/x/snapshotx"

	"github.com/stretchr/testify/require"
)

func TestSecureVM(t *testing.T) {
	for i, contents := range []string{
		"local contents = importstr 'jsonnet.go'; { contents: contents }",
		"local contents = import 'stub/import.jsonnet'; { contents: contents }",
	} {
		t.Run(fmt.Sprintf("case=%d", i), func(t *testing.T) {
			vm := MakeSecureVM()
			result, err := vm.EvaluateAnonymousSnippet("test", contents)
			require.Error(t, err, "%s", result)

			vm = jsonnet.MakeVM()
			result, err = vm.EvaluateAnonymousSnippet("test", contents)
			require.NoError(t, err)
			snapshotx.SnapshotT(t, result)
		})
	}

	t.Run("case=importbin", func(t *testing.T) {
		// importbin does not exist in the current version, but is already merged on the main branch:
		// https://github.com/google/go-jsonnet/commit/856bd58872418eee1cede0badea5b7b462c429eb
		vm := MakeSecureVM()
		result, err := vm.EvaluateAnonymousSnippet("test", "local contents = importbin 'stub/import.jsonnet'; { contents: contents }")
		require.Error(t, err, "%s", result)
	})
}

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
}

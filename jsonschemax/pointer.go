package jsonschemax

import (
	"strings"

	"github.com/ory/x/stringsx"
)

// JSONPointerToDotNotation converts JSON Pointer "#/foo/bar" to dot-notation "foo.bar".
func JSONPointerToDotNotation(pointer string) string {
	return strings.Join(stringsx.Splitx(strings.TrimPrefix(pointer, "#/"), "/"), ".")
}

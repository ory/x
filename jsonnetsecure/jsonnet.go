package jsonnetsecure

import (
	"fmt"

	"github.com/google/go-jsonnet"
)

func MakeSecureVM() *jsonnet.VM {
	vm := jsonnet.MakeVM()
	vm.Importer(new(ErrorImporter))
	return vm
}

// ErrorImporter errors when calling "import".
type ErrorImporter struct{}

// Import fetches data from a map entry.
// All paths are treated as absolute keys.
func (importer *ErrorImporter) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	return jsonnet.Contents{}, "", fmt.Errorf("import not available %v", importedPath)
}

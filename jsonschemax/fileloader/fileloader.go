package fileloader

import (
	"io"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v2"
)

// Load implements jsonschema.Loader
func Load(url string) (io.ReadCloser, error) {
	f, err := os.Open(strings.TrimPrefix(url, "file://"))
	if err != nil {
		return nil, err
	}
	return f, nil
}

func init() {
	jsonschema.Loaders["file"] = Load
}

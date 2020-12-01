package configx

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/ory/x/watcherx"
)

// KoanfFile implements a KoanfFile provider.
type KoanfFile struct {
	path string
	ctx  context.Context
}

// Provider returns a file provider.
func NewKoanfFile(ctx context.Context, path string) *KoanfFile {
	return &KoanfFile{path: filepath.Clean(path), ctx: ctx}
}

// ReadBytes reads the contents of a file on disk and returns the bytes.
func (f *KoanfFile) ReadBytes() ([]byte, error) {
	return ioutil.ReadFile(f.path)
}

// Read is not supported by the file provider.
func (f *KoanfFile) Read() (map[string]interface{}, error) {
	return nil, errors.New("file provider does not support this method")
}

// WatchChannel watches the file and triggers a callback when it changes. It is a
// blocking function that internally spawns a goroutine to watch for changes.
func (f *KoanfFile) WatchChannel(c watcherx.EventChannel) error {
	return watcherx.WatchFile(f.ctx, f.path, c)
}

package configx

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"

	"github.com/ory/x/stringslice"

	"github.com/pkg/errors"

	"github.com/ory/x/watcherx"
)

// KoanfFile implements a KoanfFile provider.
type KoanfFile struct {
	subKey string
	path   string
	ctx    context.Context
	parser koanf.Parser
}

// Provider returns a file provider.
func NewKoanfFile(ctx context.Context, path string) (*KoanfFile, error) {
	return NewKoanfFileSubKey(ctx, path, "")
}

func NewKoanfFileSubKey(ctx context.Context, path, subKey string) (*KoanfFile, error) {
	kf := &KoanfFile{
		path:   filepath.Clean(path),
		ctx:    ctx,
		subKey: subKey,
	}

	switch e := filepath.Ext(path); e {
	case ".toml":
		kf.parser = toml.Parser()
	case ".json":
		kf.parser = json.Parser()
	case ".yaml", ".yml":
		kf.parser = yaml.Parser()
	default:
		return nil, errors.Errorf("unknown config file extension: %s", e)
	}

	return kf, nil
}

// ReadBytes reads the contents of a file on disk and returns the bytes.
func (f *KoanfFile) ReadBytes() ([]byte, error) {
	return nil, errors.New("file provider does not support this method")
}

// Read is not supported by the file provider.
func (f *KoanfFile) Read() (map[string]interface{}, error) {
	//#nosec G304 -- false positive
	fc, err := os.ReadFile(f.path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	v, err := f.parser.Unmarshal(fc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if f.subKey == "" {
		return v, nil
	}

	path := strings.Split(f.subKey, Delimiter)
	for _, k := range stringslice.Reverse(path) {
		v = map[string]interface{}{
			k: v,
		}
	}

	return v, nil
}

// WatchChannel watches the file and triggers a callback when it changes. It is a
// blocking function that internally spawns a goroutine to watch for changes.
func (f *KoanfFile) WatchChannel(c watcherx.EventChannel) (watcherx.Watcher, error) {
	return watcherx.WatchFile(f.ctx, f.path, c)
}

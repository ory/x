package fsx

import (
	"io/fs"

	"github.com/pkg/errors"
)

type merged []fs.FS

var (
	_ fs.FS = (merged)(nil)
)

// Merge multiple filesystems. Later file systems are shadowed by previous ones.
func Merge(fss ...fs.FS) fs.FS {
	return merged(fss)
}

func (m merged) Open(name string) (fs.File, error) {
	for _, fsys := range m {
		f, err := fsys.Open(name)
		if err == nil {
			return f, err
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, errors.WithStack(err)
		}
	}
	return nil, errors.WithStack(fs.ErrNotExist)
}

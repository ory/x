package watcherx

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type FileWatcher struct {
	c    chan Event
	file string
	w    *fsnotify.Watcher
}

func NewFileWatcher(ctx context.Context, file string, c chan Event) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	dir := path.Dir(file)
	if err := w.Add(dir); err != nil {
		return nil, errors.WithStack(err)
	}
	realFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			// the file watcher should still watch the directory to get notified for
			// file creation
			realFile = ""
		} else {
			return nil, errors.WithStack(err)
		}
	}
	if realFile != "" && realFile != file {
		// file is a symlink, we have to explicitly watch the referenced file.
		// We are watching file instead of lastFile because fsnotify identifies
		// watch entries by the passed name but follows symlinks when watching
		// (at least on unix but not on windows).
		if err := w.Add(file); err != nil {
			return nil, errors.WithStack(err)
		}
	}
	go streamEvents(ctx, w, c, file, realFile)
	return &FileWatcher{
		c:    c,
		file: file,
		w:    w,
	}, nil
}

func streamEvents(ctx context.Context, watcher *fsnotify.Watcher, to chan Event, file, lastFile string) {
	dispatchEvent := func(data io.Reader, err error) {
		to <- Event{
			Data:  data,
			Error: err,
			Src:   file,
		}
	}
	for {
		if lastFile != "" && lastFile != file {
			// file is a symlink, see above explanation for details
			if _, err := os.Lstat(file); err == nil {
				// the symlink itself has to exist to watch it
				// if it does not the dir watcher will notify us when it gets created
				if err := watcher.Add(file); err != nil {
					dispatchEvent(nil, errors.WithStack(err))
				}
			}

		}
		select {
		case <-ctx.Done():
			_ = watcher.Close()
			return
		case e, ok := <-watcher.Events:
			if !ok {
				_ = watcher.Close()
				return
			}
			// e.Name contains the name we started the watcher with, not the actual file name
			if path.Clean(e.Name) == file && e.Op&fsnotify.Remove != 0 {
				// the file (or the file behind the symlink) was removed
				dispatchEvent(nil, nil)
				goto loopEnd
			}
			currentFile, err := filepath.EvalSymlinks(file)
			if err != nil {
				dispatchEvent(nil, errors.WithStack(err))
				goto loopEnd
			}
			// We care about three cases:
			// 1. the file was written or created
			// 2. the file is a symlink and has changed (k8s config map updates)
			// 3. the file behind the symlink was written or created
			const writeOrCreate = fsnotify.Write | fsnotify.Create
			if (path.Clean(e.Name) == file && e.Op&writeOrCreate != 0) ||
				(currentFile != lastFile) {
				lastFile = currentFile
				data, err := ioutil.ReadFile(file)
				dispatchEvent(bytes.NewBuffer(data), errors.WithStack(err))
			}
		}
	loopEnd:
		if lastFile != file {
			// Analogously to how we add the file watcher we remove it here again.
			// In the next iteration it will be readded with the current symlink (remember that fsnotify follows symlinks).
			// We ignore the error as the file might already have been removed or the
			// watcher closed on it's own.
			_ = watcher.Remove(file)
		}
	}
}

func (f *FileWatcher) ID() string {
	return f.file
}

func (f *FileWatcher) Close() {
	_ = f.w.Close()
}

var _ Watcher = &FileWatcher{}

package watcherx

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func WatchFile(ctx context.Context, file string, c EventChannel) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.WithStack(err)
	}
	dir := path.Dir(file)
	if err := w.Add(dir); err != nil {
		return errors.WithStack(err)
	}
	resolvedFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			return errors.WithStack(err)
		}
		// The file does not exist. The watcher should still watch the directory
		// to get notified about file creation.
		resolvedFile = ""
	} else if resolvedFile != file {
		// If `resolvedFile` != `file` then `file` is a symlink and we have to explicitly watch the referenced file.
		// This is because fsnotify follows symlinks and watches the destination file, not the symlink
		// itself. That is at least the case for unix systems. See: https://github.com/fsnotify/fsnotify/issues/199
		if err := w.Add(file); err != nil {
			return errors.WithStack(err)
		}
	}
	go streamFileEvents(ctx, w, c, file, resolvedFile)
	return nil
}

// streamFileEvents watches for file changes and supports symlinks which requires several workarounds due to limitations of fsnotify.
// Argument `resolvedFile` is the resolved symlink path of the file, or it is the watchedFile name itself. If `resolvedFile` is empty, then the watchedFile does not exist.
func streamFileEvents(ctx context.Context, watcher *fsnotify.Watcher, c EventChannel, watchedFile, resolvedFile string) {
	eventSource := source(watchedFile)
	removeDirectFileWatcher := func() {
		_ = watcher.Remove(watchedFile)
	}
	addDirectFileWatcher := func() {
		// check if the watchedFile (symlink) exists
		// if it does not the dir watcher will notify us when it gets created
		if _, err := os.Lstat(watchedFile); err == nil {
			if err := watcher.Add(watchedFile); err != nil {
				c <- &ErrorEvent{
					error:  errors.WithStack(err),
					source: eventSource,
				}
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			_ = watcher.Close()
			return
		case e, ok := <-watcher.Events:
			if !ok {
				_ = watcher.Close()
				return
			}
			// filter events to only watch watchedFile
			// e.Name contains the name of the watchedFile (regardless whether it is a symlink), not the resolved file name
			if path.Clean(e.Name) == watchedFile {
				if e.Op&fsnotify.Remove != 0 {
					// the watchedFile (or the file behind the symlink) was removed
					c <- &RemoveEvent{eventSource}
					removeDirectFileWatcher()
					continue
				}
				// from now on we assume watchedFile exists as there was an event on it but it was not fsnotify.Remove
				recentlyResolvedFile, err := filepath.EvalSymlinks(watchedFile)
				if err != nil {
					c <- &ErrorEvent{
						error:  errors.WithStack(err),
						source: eventSource,
					}
					continue
				}
				// This catches following three cases:
				// 1. the watchedFile was written or created
				// 2. the watchedFile is a symlink and has changed (k8s config map updates)
				// 3. the watchedFile behind the symlink was written or created
				switch {
				case recentlyResolvedFile != resolvedFile:
					resolvedFile = recentlyResolvedFile
					// watch the symlink again to update the actually watched file
					removeDirectFileWatcher()
					addDirectFileWatcher()
					// we fallthrough because we also want to read the file in this case
					fallthrough
				case e.Op&(fsnotify.Write|fsnotify.Create) != 0:
					data, err := ioutil.ReadFile(watchedFile)
					if err != nil {
						c <- &ErrorEvent{
							error:  errors.WithStack(err),
							source: eventSource,
						}
						continue
					}
					c <- &ChangeEvent{
						data:   bytes.NewBuffer(data),
						source: eventSource,
					}
				}
			}
		}
	}
}

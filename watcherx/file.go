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
	realFile, err := filepath.EvalSymlinks(file)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			// the file watcher should still watch the directory to get notified for
			// file creation
			realFile = ""
		} else {
			return errors.WithStack(err)
		}
	}
	if realFile != "" && realFile != file {
		// file is a symlink, we have to explicitly watch the referenced file.
		// We are watching file instead of lastFile because fsnotify identifies
		// watch entries by the passed name but follows symlinks when watching
		// (at least on unix but not on windows).
		if err := w.Add(file); err != nil {
			return errors.WithStack(err)
		}
	}
	go streamFileEvents(ctx, w, c, file, realFile)
	return nil
}

func streamFileEvents(ctx context.Context, watcher *fsnotify.Watcher, c EventChannel, file, lastFile string) {
	eventSource := source(file)
	for {
		if lastFile != "" && lastFile != file {
			// file is a symlink, see above explanation for details
			if _, err := os.Lstat(file); err == nil {
				// the symlink itself has to exist to watch it
				// if it does not the dir watcher will notify us when it gets created
				if err := watcher.Add(file); err != nil {
					c <- &ErrorEvent{
						error:  errors.WithStack(err),
						source: eventSource,
					}
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
				c <- &RemoveEvent{eventSource}
				goto loopEnd
			}
			currentFile, err := filepath.EvalSymlinks(file)
			if err != nil {
				c <- &ErrorEvent{
					error:  errors.WithStack(err),
					source: eventSource,
				}
				goto loopEnd
			}
			// We care about three cases:
			// 1. the file was written or created
			// 2. the file is a symlink and has changed (k8s config map updates)
			// 3. the file behind the symlink was written or created

			if (path.Clean(e.Name) == file && e.Op&(fsnotify.Write|fsnotify.Create) != 0) ||
				(currentFile != lastFile) {
				lastFile = currentFile
				data, err := ioutil.ReadFile(file)
				if err != nil {
					c <- &ErrorEvent{
						error:  errors.WithStack(err),
						source: eventSource,
					}
				} else {
					c <- &ChangeEvent{
						data:   bytes.NewBuffer(data),
						source: eventSource,
					}
				}
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

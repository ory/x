package watcherx

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func listSubDirsDepth(parent string, depth int) (dirs []string, errs []error) {
	if depth == 0 {
		return
	}
	entries, err := ioutil.ReadDir(parent)
	if err != nil {
		return dirs, append(errs, errors.WithStack(err))
	}
	for _, e := range entries {
		if e.IsDir() {
			dn := path.Join(parent, e.Name())
			subDirs, subErrs := listSubDirsDepth(dn, depth-1)
			dirs = append(append(dirs, dn), subDirs...)
			errs = append(errs, subErrs...)
		}
	}
	return
}

func WatchDirectory(ctx context.Context, dir string, depth int, c EventChannel) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.WithStack(err)
	}
	subDirs, errs := listSubDirsDepth(dir, depth)
	if len(errs) != 0 {
		return errors.Errorf("%+v", errs)
	}
	for _, d := range append(subDirs, dir) {
		fmt.Printf("adding watcher for %s\n", d)
		if err := w.Add(d); err != nil {
			return errors.WithStack(err)
		}
	}
	go streamDirectoryEvents(ctx, w, c)
	return nil
}

func handleEvent(e fsnotify.Event, w *fsnotify.Watcher, c EventChannel) {
	if e.Op&fsnotify.Remove != 0 {
		// We cannot figure out anymore if it was a file or directory.
		// If it was a directory it was added to the watchers as well as it's parent.
		// Therefore we will get two consecutive remove events from inotify (REMOVE and REMOVE_SELF).
		// Sometimes the second event has an empty name (no specific reason for that).
		// If there is no second event (timeout 1ms) we assume it was a file that got deleted.
		// This means that file deletion events are delayed by 1ms.
		select {
		case <-time.After(time.Millisecond):
			c <- &RemoveEvent{
				source: source(e.Name),
			}
			return
		case secondE := <-w.Events:
			if (secondE.Name != "" && secondE.Name != e.Name) || secondE.Op&fsnotify.Remove == 0 {
				// this is NOT the unix.IN_DELETE_SELF event => we have to handle the first explicitly
				// and the second recursively because it might be the first event of a directory deletion
				c <- &RemoveEvent{
					source: source(e.Name),
				}
				handleEvent(secondE, w, c)
			} // else we do not want any event on deletion of a folder
		}
	} else if e.Op&(fsnotify.Write|fsnotify.Create) != 0 {
		if stats, err := os.Stat(e.Name); err != nil {
			c <- &ErrorEvent{
				error:  errors.WithStack(err),
				source: source(e.Name),
			}
			return
		} else if stats.IsDir() {
			fmt.Printf("created directory %s\n", e.Name)
			if err := w.Add(e.Name); err != nil {
				c <- &ErrorEvent{
					error:  errors.WithStack(err),
					source: source(e.Name),
				}
			}
			return
		}
		data, err := ioutil.ReadFile(e.Name)
		if err != nil {
			c <- &ErrorEvent{
				error:  err,
				source: source(e.Name),
			}
		} else {
			c <- &ChangeEvent{
				data:   bytes.NewBuffer(data),
				source: source(e.Name),
			}
		}
	}
}

func streamDirectoryEvents(ctx context.Context, w *fsnotify.Watcher, c EventChannel) {
	for {
		select {
		case <-ctx.Done():
			_ = w.Close()
			return
		case e := <-w.Events:
			handleEvent(e, w, c)
		}
	}
}

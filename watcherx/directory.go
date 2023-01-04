// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package watcherx

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

func WatchDirectory(ctx context.Context, dir string, c EventChannel) (Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var subDirs []string
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			subDirs = append(subDirs, path)
		}
		return nil
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	for _, d := range append(subDirs, dir) {
		if err := w.Add(d); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	d := newDispatcher()
	go streamDirectoryEvents(ctx, w, c, d.trigger, d.done, dir)
	return d, nil
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
			if err := w.Add(e.Name); err != nil {
				c <- &ErrorEvent{
					error:  errors.WithStack(err),
					source: source(e.Name),
				}
			}
			return
		}

		//#nosec G304 -- false positive
		data, err := ioutil.ReadFile(e.Name)
		if err != nil {
			c <- &ErrorEvent{
				error:  err,
				source: source(e.Name),
			}
		} else {
			c <- &ChangeEvent{
				data:   data,
				source: source(e.Name),
			}
		}
	}
}

func streamDirectoryEvents(ctx context.Context, w *fsnotify.Watcher, c EventChannel, sendNow <-chan struct{}, sendNowDone chan<- int, dir string) {
	for {
		select {
		case <-ctx.Done():
			_ = w.Close()
			return
		case e := <-w.Events:
			handleEvent(e, w, c)
		case <-sendNow:
			var eventsSent int

			if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					//#nosec G304 -- false positive
					data, err := ioutil.ReadFile(path)
					if err != nil {
						c <- &ErrorEvent{
							error:  err,
							source: source(path),
						}
					} else {
						c <- &ChangeEvent{
							data:   data,
							source: source(path),
						}
					}
					eventsSent++
				}
				return nil
			}); err != nil {
				c <- &ErrorEvent{
					error:  err,
					source: source(dir),
				}
				eventsSent++
			}

			sendNowDone <- eventsSent
		}
	}
}

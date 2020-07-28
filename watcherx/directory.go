package watcherx

import (
	"bytes"
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
)

func WatchDirectory(ctx context.Context, dir string, c EventChannel) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.WithStack(err)
	}
	if err := w.Add(dir); err != nil {
		return errors.WithStack(err)
	}
	go streamDirectoryEvents(ctx, w, dir, c)
	return nil
}

func streamDirectoryEvents(ctx context.Context, w *fsnotify.Watcher, dir string, c EventChannel) {
	for {
		select {
		case <-ctx.Done():
			_ = w.Close()
			return
		case e := <-w.Events:
			fmt.Printf("got event %s\n", e.String())
			if e.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if stats, err := os.Stat(e.Name); err != nil {
					c <- &ErrorEvent{
						error:  errors.WithStack(err),
						source: source(e.Name),
					}
					continue
				} else if stats.IsDir() {
					fmt.Printf("created directory %s\n", e.Name)
					if err := w.Add(e.Name); err != nil {
						c <- &ErrorEvent{
							error:  errors.WithStack(err),
							source: source(e.Name),
						}
					}
					continue
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
			} else if e.Op&fsnotify.Remove != 0 {
				c <- &RemoveEvent{
					source: source(e.Name),
				}
			}
		}
	}
}

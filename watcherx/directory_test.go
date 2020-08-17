package watcherx

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatchDirectory(t *testing.T) {
	t.Run("case=notifies about file creation in directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		require.NoError(t, WatchDirectory(ctx, dir, c))
		fileName := path.Join(dir, "example")
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", fileName)
	})

	t.Run("case=notifies about file write in directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		fileName := path.Join(dir, "example")
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, WatchDirectory(ctx, dir, c))

		_, err = fmt.Fprintf(f, "content")
		require.NoError(t, f.Close())

		assertChange(t, <-c, "content", fileName)
	})

	t.Run("case=nofifies about file delete in directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		fileName := path.Join(dir, "example")
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		require.NoError(t, WatchDirectory(ctx, dir, c))
		require.NoError(t, os.Remove(fileName))

		assertRemove(t, <-c, fileName)
	})

	t.Run("case=notifies about file in child directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		childDir := path.Join(dir, "child")
		require.NoError(t, os.Mkdir(childDir, 0777))

		require.NoError(t, WatchDirectory(ctx, dir, c))

		fileName := path.Join(childDir, "example")
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", fileName)
	})

	t.Run("case=watches new child directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		require.NoError(t, WatchDirectory(ctx, dir, c))

		childDir := path.Join(dir, "child")
		require.NoError(t, os.Mkdir(childDir, 0777))
		fileName := path.Join(childDir, "example")
		// there's not much we can do about this timeout as it takes some time until the new watcher is created
		time.Sleep(time.Millisecond)
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", fileName)
	})

	t.Run("case=does not notify on directory deletion", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		childDir := path.Join(dir, "child")
		require.NoError(t, os.Mkdir(childDir, 0777))

		require.NoError(t, WatchDirectory(ctx, dir, c))

		require.NoError(t, os.Remove(childDir))

		select {
		case e := <-c:
			t.Logf("got unexpected event %T: %+v", e, e)
			t.FailNow()
		case <-time.After(2 * time.Millisecond):
			// expected to not receive an event (1ms is what the watcher waits for the second event)
		}
	})

	t.Run("case=notifies only for files on batch delete", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		childDir := path.Join(dir, "child")
		subChildDir := path.Join(childDir, "subchild")
		require.NoError(t, os.MkdirAll(subChildDir, 0777))
		f1 := path.Join(subChildDir, "f1")
		f, err := os.Create(f1)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		f2 := path.Join(childDir, "f2")
		f, err = os.Create(f2)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		require.NoError(t, WatchDirectory(ctx, dir, c))

		require.NoError(t, os.RemoveAll(childDir))

		events := []Event{<-c, <-c}
		if events[0].Source() > events[1].Source() {
			events[1], events[0] = events[0], events[1]
		}
		assertRemove(t, events[0], f2)
		assertRemove(t, events[1], f1)
	})
}

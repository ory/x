package watcherx

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
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
}

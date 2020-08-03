package watcherx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/stringslice"
)

func TestListSubDirsDepth(t *testing.T) {
	absSetup := func(t *testing.T) (string, []string) {
		wd, err := ioutil.TempDir("", "list-sub-dir-test-")
		var dirs []string
		for _, d := range []string{
			"bar",
			"bar/baz",
			"foo",
		} {
			dirs = append(dirs, path.Join(wd, d))
		}
		require.NoError(t, err)

		for _, d := range dirs {
			require.NoError(t, os.MkdirAll(d, 0777))
		}

		return wd, dirs
	}

	relSetup := func(t *testing.T) (func(), []string) {
		dirs := []string{
			"bar",
			"bar/baz",
			"foo",
		}
		wd, err := ioutil.TempDir("", "list-sub-dir-test-")
		require.NoError(t, err)

		for _, d := range dirs {
			require.NoError(t, os.MkdirAll(path.Join(wd, d), 0777))
		}

		prevWD, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(wd))
		return func() {
			require.NoError(t, os.Chdir(prevWD))
		}, dirs
	}

	t.Run("case=absolute path", func(t *testing.T) {
		wd, dirs := absSetup(t)

		subDirs, errs := listSubDirsDepth(wd, 1000)
		require.Equal(t, 0, len(errs), "%+v", errs)
		assert.Equal(t, dirs, subDirs)
	})

	t.Run("case=relative path", func(t *testing.T) {
		clean, dirs := relSetup(t)
		defer clean()

		subDirs, errs := listSubDirsDepth(".", 100)

		require.Equal(t, 0, len(errs), "%+v", errs)
		assert.Equal(t, dirs, subDirs)
	})

	t.Run("case=respects depth", func(t *testing.T) {
		clean, dirs := relSetup(t)
		defer clean()

		subDirs, errs := listSubDirsDepth(".", 1)
		require.Equal(t, 0, len(errs), "%+v", errs)
		assert.Equal(t, stringslice.Filter(dirs, func(s string) bool {
			return len(strings.Split(s, string(os.PathSeparator))) > 1
		}), subDirs)
	})
}

func TestWatchDirectory(t *testing.T) {
	t.Run("case=notifies about file creation in directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))
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
		require.NoError(t, WatchDirectory(ctx, dir, 100, c))

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

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))
		require.NoError(t, os.Remove(fileName))

		assertRemove(t, <-c, fileName)
	})

	t.Run("case=notifies about file in child directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		childDir := path.Join(dir, "child")
		require.NoError(t, os.Mkdir(childDir, 0777))

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))

		fileName := path.Join(childDir, "example")
		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", fileName)
	})

	t.Run("case=watches new child directory", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))

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

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))

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

		require.NoError(t, WatchDirectory(ctx, dir, 100, c))

		require.NoError(t, os.RemoveAll(childDir))

		assertRemove(t, <-c, f2)
		assertRemove(t, <-c, f1)
	})
}

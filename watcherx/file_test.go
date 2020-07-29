package watcherx

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (context.Context, chan Event, string, context.CancelFunc) {
	c := make(chan Event)
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := ioutil.TempDir("", "*")
	require.NoError(t, err)
	return ctx, c, dir, cancel
}

func assertChange(t *testing.T, e Event, expectedData, src string) {
	_, ok := e.(*ChangeEvent)
	assert.True(t, ok)
	data, err := ioutil.ReadAll(e.Reader())
	require.NoError(t, err)
	assert.Equal(t, expectedData, string(data))
	assert.Equal(t, src, e.Source())
}

func assertRemove(t *testing.T, e Event, src string) {
	assert.Equal(t, &RemoveEvent{source(src)}, e)
}

func TestFileWatcher(t *testing.T) {
	t.Run("case=notifies on file write", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		f, err := os.Create(exampleFile)
		require.NoError(t, err)

		require.NoError(t, WatchFile(ctx, exampleFile, c))

		_, err = fmt.Fprintf(f, "foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "foo", exampleFile)
	})

	t.Run("case=notifies on file create", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		require.NoError(t, WatchFile(ctx, exampleFile, c))

		f, err := os.Create(exampleFile)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", exampleFile)
	})

	t.Run("case=notifies after file delete about recreate", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		f, err := os.Create(exampleFile)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		require.NoError(t, WatchFile(ctx, exampleFile, c))

		require.NoError(t, os.Remove(exampleFile))

		assertRemove(t, <-c, exampleFile)

		f, err = os.Create(exampleFile)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", exampleFile)
	})

	t.Run("case=notifies about changes in the linked file", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		otherDir, err := ioutil.TempDir("", "*")
		require.NoError(t, err)
		origFileName := path.Join(otherDir, "original")
		f, err := os.Create(origFileName)
		require.NoError(t, err)

		linkFileName := path.Join(dir, "slink")
		require.NoError(t, os.Symlink(origFileName, linkFileName))

		require.NoError(t, WatchFile(ctx, linkFileName, c))

		_, err = fmt.Fprintf(f, "content")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "content", linkFileName)
	})

	t.Run("case=notifies about symlink change", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		otherDir, err := ioutil.TempDir("", "*")
		require.NoError(t, err)
		fileOne := path.Join(otherDir, "fileOne")
		fileTwo := path.Join(otherDir, "fileTwo")
		f1, err := os.Create(fileOne)
		require.NoError(t, err)
		require.NoError(t, f1.Close())
		f2, err := os.Create(fileTwo)
		require.NoError(t, err)
		_, err = fmt.Fprintf(f2, "file two")
		require.NoError(t, err)
		require.NoError(t, f2.Close())

		linkFileName := path.Join(dir, "slink")
		require.NoError(t, os.Symlink(fileOne, linkFileName))

		require.NoError(t, WatchFile(ctx, linkFileName, c))

		require.NoError(t, os.Remove(linkFileName))
		assertRemove(t, <-c, linkFileName)

		require.NoError(t, os.Symlink(fileTwo, linkFileName))
		assertChange(t, <-c, "file two", linkFileName)
	})

	t.Run("case=watch relative file path", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		require.NoError(t, os.Chdir(dir))

		fileName := "example.file"
		require.NoError(t, WatchFile(ctx, fileName, c))

		f, err := os.Create(fileName)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c, "", fileName)
	})
}

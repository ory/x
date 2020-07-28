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

	"github.com/ory/x/pointerx"
	"github.com/ory/x/randx"
)

func createTmpDir(t *testing.T) string {
	dir := path.Join(os.TempDir(), randx.MustString(32, randx.AlphaNum))
	require.NoError(t, os.Mkdir(dir, 0777))
	return dir
}

func setup(t *testing.T) (context.Context, chan Event, string, context.CancelFunc) {
	c := make(chan Event)
	ctx, cancel := context.WithCancel(context.Background())
	dir := createTmpDir(t)
	return ctx, c, dir, cancel
}

func assertEvent(t *testing.T, expectedError error, expectedSrc string, expectedData *string, c chan Event) {
	select {
	case e := <-c:
		assert.Equal(t, expectedError, e.Error, "%+v", e.Error)
		assert.Equal(t, expectedSrc, e.Src)
		if expectedData != nil {
			require.NotNil(t, e.Data)
			data, err := ioutil.ReadAll(e.Data)
			require.NoError(t, err)
			assert.Equal(t, *expectedData, string(data))
		} else {
			assert.Nil(t, e.Data)
		}
	}
}

func TestFileWatcher(t *testing.T) {
	t.Run("case=notifies on file write", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		f, err := os.Create(exampleFile)
		require.NoError(t, err)

		_, err = NewFileWatcher(ctx, exampleFile, c)
		require.NoError(t, err)

		_, err = fmt.Fprintf(f, "foo")
		require.NoError(t, err)

		assertEvent(t, nil, exampleFile, pointerx.String("foo"), c)
	})

	t.Run("case=notifies on file create", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		_, err := NewFileWatcher(ctx, exampleFile, c)
		require.NoError(t, err)

		_, err = os.Create(exampleFile)
		require.NoError(t, err)

		assertEvent(t, nil, exampleFile, pointerx.String(""), c)
	})

	t.Run("case=notifies after file delete about recreate", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		exampleFile := path.Join(dir, "example.file")
		f, err := os.Create(exampleFile)
		require.NoError(t, err)

		_, err = NewFileWatcher(ctx, exampleFile, c)
		require.NoError(t, err)

		require.NoError(t, f.Close())
		require.NoError(t, os.Remove(exampleFile))

		assertEvent(t, nil, exampleFile, nil, c)

		f, err = os.Create(exampleFile)
		defer f.Close()
		require.NoError(t, err)

		assertEvent(t, nil, exampleFile, pointerx.String(""), c)
	})

	t.Run("case=notifies about changes in the linked file", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		otherDir := createTmpDir(t)
		origFileName := path.Join(otherDir, "original")
		f, err := os.Create(origFileName)
		defer f.Close()
		require.NoError(t, err)

		linkFileName := path.Join(dir, "slink")
		require.NoError(t, os.Symlink(origFileName, linkFileName))

		_, err = NewFileWatcher(ctx, linkFileName, c)
		require.NoError(t, err)

		_, err = fmt.Fprintf(f, "content")
		require.NoError(t, err)

		assertEvent(t, nil, linkFileName, pointerx.String("content"), c)
	})

	t.Run("case=notifies about symlink change", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()

		otherDir := createTmpDir(t)
		fileOne := path.Join(otherDir, "fileOne")
		fileTwo := path.Join(otherDir, "fileTwo")
		f1, err := os.Create(fileOne)
		defer f1.Close()
		require.NoError(t, err)
		f2, err := os.Create(fileTwo)
		defer f2.Close()
		require.NoError(t, err)
		_, err = fmt.Fprintf(f2, "file two")
		require.NoError(t, err)

		linkFileName := path.Join(dir, "slink")
		require.NoError(t, os.Symlink(fileOne, linkFileName))

		_, err = NewFileWatcher(ctx, linkFileName, c)
		require.NoError(t, err)

		require.NoError(t, os.Remove(linkFileName))
		t.Logf("working in dir %s otherdir %s", dir, otherDir)
		assertEvent(t, nil, linkFileName, nil, c)

		require.NoError(t, os.Symlink(fileTwo, linkFileName))
		assertEvent(t, nil, linkFileName, pointerx.String("file two"), c)
	})
}

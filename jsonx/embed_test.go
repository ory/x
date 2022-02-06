package jsonx

import (
	"github.com/ory/x/snapshotx"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbedSources(t *testing.T) {
	t.Run("fixtures", func(t *testing.T) {
		require.NoError(t, filepath.Walk("fixture/embed", func(p string, i fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if i.IsDir() {
				return nil
			}

			t.Run("fixture="+i.Name(), func(t *testing.T) {
				t.Parallel()

				input, err := os.ReadFile(p)
				require.NoError(t, err)

				actual, err := EmbedSources(input)
				require.NoError(t, err)

				snapshotx.SnapshotTExcept(t, actual, nil)
			})

			return nil
		}))
	})

	t.Run("fails on invalid source", func(t *testing.T) {
		_, err := EmbedSources([]byte(`{"foo":"base64://invalid"}`))
		require.Error(t, err)
	})
}

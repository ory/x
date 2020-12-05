package configx

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKoanfFile(t *testing.T) {
	setupFile := func(t *testing.T, fn, fc, subKey string) *KoanfFile {
		f, err := ioutil.TempFile("", fn)
		require.NoError(t, err)
		_, err = f.Write([]byte(fc))
		require.NoError(t, err)

		kf, err := NewKoanfFileSubKey(context.Background(), f.Name(), subKey)
		require.NoError(t, err)
		return kf
	}

	t.Run("case=reads json root file", func(t *testing.T) {
		v := map[string]interface{}{
			"foo": "bar",
		}
		encV, err := json.Marshal(v)
		require.NoError(t, err)

		kf := setupFile(t, "config*.json", string(encV), "")

		actual, err := kf.Read()
		require.NoError(t, err)
		assert.Equal(t, v, actual)
	})

	t.Run("case=reads yaml root file", func(t *testing.T) {
		v := map[string]interface{}{
			"foo": "yaml string",
		}
		encV, err := yaml.Marshal(v)
		require.NoError(t, err)

		kf := setupFile(t, "config*.yml", string(encV), "")

		actual, err := kf.Read()
		require.NoError(t, err)
		assert.Equal(t, v, actual)
	})

	t.Run("case=reads toml root file", func(t *testing.T) {
		v := map[string]interface{}{
			"foo": "toml string",
		}
		encV, err := toml.Marshal(v)
		require.NoError(t, err)

		kf := setupFile(t, "config*.toml", string(encV), "")

		actual, err := kf.Read()
		require.NoError(t, err)
		assert.Equal(t, v, actual)
	})

	t.Run("case=reads json file as subkey", func(t *testing.T) {
		v := map[string]interface{}{
			"bar": "asdf",
		}
		encV, err := json.Marshal(v)
		require.NoError(t, err)

		kf := setupFile(t, "config*.json", string(encV), "parent.of.config")

		actual, err := kf.Read()
		require.NoError(t, err)
		assert.Equal(t, map[string]interface{}{
			"parent": map[string]interface{}{
				"of": map[string]interface{}{
					"config": v,
				},
			},
		}, actual)
	})
}

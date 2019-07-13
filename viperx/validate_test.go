package viperx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/gojsonschema"
)

func TestValidate(t *testing.T) {
	path, err := filepath.Abs("./stub/config.schema.json")
	require.NoError(t, err)

	loader := gojsonschema.NewReferenceLoader("file://" + strings.Replace(path, "\\", "/", -1))

	t.Run("case=set", func(t *testing.T) {
		viper.Reset()

		viper.Set("dsn", "memory")
		InitializeConfig(uuid.New().String(), "", nil)

		require.NoError(t, Validate(loader))
	})

	t.Run("case=missing-dsn", func(t *testing.T) {
		viper.Reset()

		InitializeConfig(uuid.New().String(), "", nil)

		err := Validate(loader)
		require.Error(t, Validate(loader))
		assert.Contains(t, err.Error(), "dsn")
	})

	t.Run("case=env", func(t *testing.T) {
		viper.Reset()

		require.NoError(t, os.Setenv("DSN", "memory"))
		require.NoError(t, viper.BindEnv("dsn"))

		InitializeConfig(uuid.New().String(), "", nil)

		require.NoError(t, Validate(loader))
		require.NoError(t, os.Setenv("DSN", ""))
	})

	t.Run("case=file", func(t *testing.T) {
		viper.Reset()

		InitializeConfig("config", "stub", nil)

		require.NoError(t, Validate(loader))
	})
}

func TestToMapStringInterface(t *testing.T) {
	assert.EqualValues(
		t,
		map[string]interface{}{
			"foo": "bar",
			"items": map[string]interface{}{
				"foo": "bar",
			},
		},
		toMapStringInterface(map[string]interface{}{
			"foo": "bar",
			"items": map[interface{}]interface{}{
				"foo": "bar",
			},
		}),
	)
}

package viperx

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ory/x/logrusx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
	_ "github.com/ory/jsonschema/v3/fileloader"

	"github.com/ory/viper"
)

func TestValidate(t *testing.T) {
	l := logrusx.New("test", "testing")
	path, err := filepath.Abs("./stub/config.schema.json")
	require.NoError(t, err)

	file, err := jsonschema.LoadURL("file://" + strings.Replace(path, "\\", "/", -1))
	require.NoError(t, err)

	schema, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	t.Run("case=set", func(t *testing.T) {
		viper.Reset()

		viper.Set("dsn", "memory")
		InitializeConfig(uuid.New().String(), "", nil)

		require.NoError(t, Validate(l, "schema.json", schema))
	})

	t.Run("case=missing-dsn", func(t *testing.T) {
		viper.Reset()

		InitializeConfig(uuid.New().String(), "", nil)

		require.Error(t, Validate(l, "schema.json", schema))
	})

	t.Run("case=env", func(t *testing.T) {
		viper.Reset()

		require.NoError(t, os.Setenv("DSN", "memory"))
		InitializeConfig(uuid.New().String(), "", nil)

		require.NoError(t, Validate(l, "schema.json", schema))
		require.NoError(t, os.Setenv("DSN", ""))
	})

	t.Run("case=file", func(t *testing.T) {
		viper.Reset()

		InitializeConfig("config", "stub", nil)

		require.NoError(t, Validate(l, "schema.json", schema))
	})

	t.Run("case=ValidateFromURL", func(t *testing.T) {
		viper.Reset()

		InitializeConfig("config", "stub", nil)

		require.NoError(t, ValidateFromURL(l, path))
	})
}

func TestLoggerWithValidationErrorFields(t *testing.T) {
	l := logrusx.New("test", "testing")
	t.Run("case=required", func(t *testing.T) {
		viper.Reset()

		err := ValidateFromURL(l, "file://stub/config.schema.json")
		require.Error(t, err)

		var b bytes.Buffer
		PrintHumanReadableValidationErrors(&b, err)
		assert.Contains(t, b.String(), "one or more required properties are missing")
	})

	t.Run("case=type", func(t *testing.T) {
		viper.Reset()

		viper.Set("dsn", 1234)
		err := ValidateFromURL(l, "file://stub/config.schema.json")
		require.Error(t, err)

		var b bytes.Buffer
		PrintHumanReadableValidationErrors(&b, err)
		assert.Contains(t, b.String(), "expected string, but got number")
	})

	t.Run("case=multiple errors", func(t *testing.T) {
		viper.Reset()

		viper.Set("dsn", 1234)
		viper.Set("foo", 1234)
		err := ValidateFromURL(l, "file://stub/config.schema.json")
		require.Error(t, err)

		expected := []struct {
			key string
			err string
		}{
			{
				key: "dsn",
				err: "expected string, but got number",
			},
			{
				key: "foo",
				err: "value must be \"bar\"",
			},
		}

		var b bytes.Buffer
		PrintHumanReadableValidationErrors(&b, err)
		jsonString := b.String()
		for _, e := range expected {
			assert.Contains(t, b.String(), e.err)
			assert.Contains(t, b.String(), e.key, "%s", jsonString)
		}
	})
}

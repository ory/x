package viperx

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/ory/jsonschema/v3"
	_ "github.com/ory/jsonschema/v3/fileloader"

	"github.com/ory/viper"
)

func TestValidate(t *testing.T) {
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

		require.NoError(t, Validate("schema.json", schema))
	})

	t.Run("case=missing-dsn", func(t *testing.T) {
		viper.Reset()

		InitializeConfig(uuid.New().String(), "", nil)

		require.Error(t, Validate("schema.json", schema))
	})

	t.Run("case=env", func(t *testing.T) {
		viper.Reset()

		require.NoError(t, os.Setenv("DSN", "memory"))
		InitializeConfig(uuid.New().String(), "", nil)

		require.NoError(t, Validate("schema.json", schema))
		require.NoError(t, os.Setenv("DSN", ""))
	})

	t.Run("case=file", func(t *testing.T) {
		viper.Reset()

		InitializeConfig("config", "stub", nil)

		require.NoError(t, Validate("schema.json", schema))
	})

	t.Run("case=ValidateFromURL", func(t *testing.T) {
		viper.Reset()

		InitializeConfig("config", "stub", nil)

		require.NoError(t, ValidateFromURL(path))
	})
}

func TestLoggerWithValidationErrorFields(t *testing.T) {
	nl := func() (logrus.FieldLogger, *bytes.Buffer) {
		var buffer bytes.Buffer
		logger := logrus.New()
		logger.Out = &buffer
		logger.Formatter = new(logrus.JSONFormatter)
		return logger, &buffer
	}

	t.Run("case=required", func(t *testing.T) {
		viper.Reset()
		l, buf := nl()

		err := ValidateFromURL("file://stub/config.schema.json")
		require.Error(t, err)

		LoggerWithValidationErrorFields(l, err).WithError(err).Print("")
		assert.EqualValues(t, "dsn", gjson.Get(buf.String(), "config_key").String(), "%s", buf.String())
		assert.EqualValues(t, "one or more required properties are missing", gjson.Get(buf.String(), "validation_error").String(), "%s", buf.String())
	})

	t.Run("case=required", func(t *testing.T) {
		viper.Reset()
		l, buf := nl()

		viper.Set("dsn", 1234)
		err := ValidateFromURL("file://stub/config.schema.json")
		require.Error(t, err)

		LoggerWithValidationErrorFields(l, err).WithError(err).Print("")
		assert.EqualValues(t, "dsn", gjson.Get(buf.String(), "config_key").String(), "%s", buf.String())
		assert.EqualValues(t, "expected string, but got number", gjson.Get(buf.String(), "validation_error").String(), "%s", buf.String())
	})
}

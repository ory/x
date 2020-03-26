package viperx

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/viper"
)

func TestBindEnv(t *testing.T) {
	readFile := func(path string) string {
		schema, err := ioutil.ReadFile(path)
		require.NoError(t, err)
		return string(schema)
	}

	t.Run("func=BindEnvsToSchema", func(t *testing.T) {
		viper.Reset()

		require.NoError(t, os.Setenv("MUTATORS_ID_TOKEN_CONFIG_JWKS_URL", "foo"))
		require.NoError(t, os.Setenv("MUTATORS_NOOP_ENABLED", "true"))

		require.NoError(t, BindEnvsToSchema([]byte(readFile("./stub/.oathkeeper.schema.json"))))

		require.NoError(t, os.Setenv("AUTHENTICATORS_COOKIE_SESSION_CONFIG_ONLY", "bar"))

		assert.Equal(t, true, viper.Get("mutators.noop.enabled"))
		assert.Equal(t, "foo", viper.Get("mutators.id_token.config.jwks_url"))
		assert.Equal(t, []string{"bar"}, viper.Get("authenticators.cookie_session.config.only"))
	})

	t.Run("case=string slice", func(t *testing.T) {
		viper.Reset()

		schema := []byte(
			`{
"type": "object",
"properties": {
  "some_strings": {
    "type": "array",
    "items": {
      "type": "string"
    }
  }
}
}`,
		)

		require.NoError(t, os.Setenv("SOME_STRINGS", "a,b,c"))

		require.NoError(t, BindEnvsToSchema(schema))

		assert.Equal(t, []string{"a", "b", "c"}, viper.Get("SOME_STRINGS"))

		require.NoError(t, os.Unsetenv("SOME_STRINGS"))
	})

	t.Run("case=string slice with default", func(t *testing.T) {
		viper.Reset()

		schema := []byte(
			`{
"type": "object",
"properties": {
  "some_strings": {
    "type": "array",
    "items": {
      "type": "string"
    },
    "default": ["foo"]
  }
}
}`,
		)

		require.NoError(t, BindEnvsToSchema(schema))

		assert.Equal(t, []string{"foo"}, viper.Get("SOME_STRINGS"))
	})

	t.Run("case=int slice with default", func(t *testing.T) {
		viper.Reset()

		schema := []byte(
			`{
"type": "object",
"properties": {
  "some_ints": {
    "type": "array",
    "items": {
      "type": "integer"
    },
    "default": [1, 2]
  }
}
}`,
		)

		require.NoError(t, BindEnvsToSchema(schema))

		assert.Equal(t, []float64{1, 2}, viper.Get("some_ints"))
	})

	t.Run("case=string slice without default or value", func(t *testing.T) {
		viper.Reset()

		schema := []byte(
			`{
"type": "object",
"properties": {
  "some_strings": {
    "type": "array",
    "items": {
      "type": "string"
    }
  }
}
}`,
		)

		require.NoError(t, BindEnvsToSchema(schema))

		assert.Equal(t, nil, viper.Get("SOME_STRINGS"))
	})
}

package logrusx

import (
	"testing"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/ory/jsonschema/v3"
)

func TestConfigSchema(t *testing.T) {
	c := jsonschema.NewCompiler()
	require.NoError(t, AddConfigSchema(c))
	schema, err := c.Compile(ConfigSchemaID)
	require.NoError(t, err)

	logConfig := map[string]interface{}{
		"level":                 "trace",
		"format":                "json_pretty",
		"leak_sensitive_values": true,
	}

	assert.NoError(t, schema.ValidateInterface(logConfig))

	rawConfig, err := sjson.Set("{}", "log", logConfig)
	require.NoError(t, err)

	k := koanf.New(".")
	require.NoError(t, k.Load(rawbytes.Provider([]byte(rawConfig)), json.Parser()))

	l := New("foo", "bar", WithConfigurator(k))

	assert.True(t, l.leakSensitive)
	assert.Equal(t, logrus.TraceLevel, l.Logger.Level)
	assert.IsType(t, &logrus.JSONFormatter{}, l.Logger.Formatter)
}

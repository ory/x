package otelx

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/ory/jsonschema/v3"
)

const rootSchema = `{
  "properties": {
    "otelx": {
      "$ref": "%s"
    }
  }
}
`

func TestConfigSchema(t *testing.T) {
	t.Run("func=AddConfigSchema", func(t *testing.T) {
		c := jsonschema.NewCompiler()
		require.NoError(t, AddConfigSchema(c))

		conf := Config{
			ServiceName: "Ory X",
			Provider:    "jaeger",
			Providers: ProvidersConfig{
				Jaeger: JaegerConfig{
					LocalAgentAddress: "localhost:6831",
					LocalAgentHost:    "localhost",
					LocalAgentPort:    6831,
					Sampling: JaegerSampling{
						ServerURL: "http://localhost:5778/sampling",
					},
				},
			},
		}

		rawConfig, err := sjson.Set("{}", "otelx", &conf)
		require.NoError(t, err)

		require.NoError(t, c.AddResource("config", bytes.NewBufferString(fmt.Sprintf(rootSchema, ConfigSchemaID))))

		schema, err := c.Compile(context.Background(), "config")
		require.NoError(t, err)

		assert.NoError(t, schema.Validate(bytes.NewBufferString(rawConfig)))
	})

	t.Run("case=AddressTakesPrecedence", func(t *testing.T) {
		conf := Config{
			ServiceName: "Ory X",
			Provider:    "jaeger",
			Providers: ProvidersConfig{
				Jaeger: JaegerConfig{
					LocalAgentAddress: "foo:6831",
					LocalAgentHost:    "bar",
					LocalAgentPort:    6832,
					Sampling: JaegerSampling{
						ServerURL: "http://localhost:5778/sampling",
					},
				},
			},
		}

		host, port := configureHostPort(conf)
		assert.Equal(t, host, "foo")
		assert.Equal(t, port, "6831")
	})
}

package tracing

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
    "tracing": {
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
			Providers: &ProvidersConfig{
				Jaeger: &JaegerConfig{
					LocalAgentAddress: "jaeger:6831",
					Sampling: &JaegerSampling{
						Type:      "const",
						Value:     1,
						ServerURL: "https://localhost:5778/sampling",
					},
					Propagation:       "jaeger",
					MaxTagValueLength: 100,
				},
				Zipkin: &ZipkinConfig{
					ServerURL: "https://example.com",
				},
			},
		}

		rawConfig, err := sjson.Set("{}", "tracing", &conf)
		require.NoError(t, err)

		require.NoError(t, c.AddResource("config", bytes.NewBufferString(fmt.Sprintf(rootSchema, ConfigSchemaID))))

		schema, err := c.Compile(context.Background(), "config")
		require.NoError(t, err)

		assert.NoError(t, schema.Validate(bytes.NewBufferString(rawConfig)))
	})
}

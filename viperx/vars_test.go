package viperx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/viper"
)

func TestUnmarshalKey(t *testing.T) {
	viper.SetConfigFile("./stub/.unmarshal-key.yml")
	require.NoError(t, viper.ReadInConfig())

	var config struct {
		Enabled bool            `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	require.NoError(t, UnmarshalKey("selfservice.strategies.oidc", &config))

	assert.EqualValues(t, true, config.Enabled)
	assert.EqualValues(t, `{"providers":[{"client_id":"kratos-client","client_secret":"kratos-secret","id":"hydra","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]},{"client_id":"google-client","client_secret":"kratos-secret","id":"google","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]},{"client_id":"github-client","client_secret":"kratos-secret","id":"github","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]}]}`, string(config.Config))
}

func TestUnmarshalKeyWithJsonObject(t *testing.T) {
	viper.SetConfigFile("./stub/.unmarshal-key.yml")
	viper.Set("selfservice.strategies.oidc", `{"enabled":true, "config":{"providers": [{"client_id":"a-different-client","client_secret":"a-different-secret","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/a-different-file.jsonnet","provider":"github","scope":["user:email"]}]}}`)
	require.NoError(t, viper.ReadInConfig())

	var config struct {
		Enabled bool            `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	require.NoError(t, UnmarshalKey("selfservice.strategies.oidc", &config))

	assert.EqualValues(t, true, config.Enabled)
	assert.EqualValues(t, `{"providers": [{"client_id":"a-different-client","client_secret":"a-different-secret","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/a-different-file.jsonnet","provider":"github","scope":["user:email"]}]}`, string(config.Config))
}

func TestUnmarshalKeyWithInvalidJson(t *testing.T) {
	viper.SetConfigFile("./stub/.unmarshal-key.yml")
	viper.Set("selfservice.strategies.oidc", `{"enabled":true, "config":{"providers": [{"client_id":"a-different-client",`)
	require.NoError(t, viper.ReadInConfig())

	var config struct {
		Enabled bool            `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	require.Error(t, UnmarshalKey("selfservice.strategies.oidc", &config))
}

func TestUnmarshalKeyWithString(t *testing.T) {
	viper.SetConfigFile("./stub/.unmarshal-key.yml")
	viper.Set("identity.default_schema_url", `https://ory.sh`)
	require.NoError(t, viper.ReadInConfig())

	var defaultSchemaURL string
	require.NoError(t, UnmarshalKey("identity.default_schema_url", &defaultSchemaURL))

	assert.EqualValues(t, `https://ory.sh`, defaultSchemaURL)
}

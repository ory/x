package viperx

import (
	"encoding/json"
	"testing"

	"github.com/ory/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalKey(t *testing.T) {
	viper.SetConfigFile("./stub/.unmarshal-key.yml")
	require.NoError(t, viper.ReadInConfig())

	var config struct {
		Enabled bool          `json:"enabled"`
		Config  json.RawMessage `json:"config"`
	}
	require.NoError(t, UnmarshalKey("selfservice.strategies.oidc", &config))

	assert.EqualValues(t, true, config.Enabled)
	assert.EqualValues(t, `{"providers":[{"client_id":"kratos-client","client_secret":"kratos-secret","id":"hydra","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]},{"client_id":"google-client","client_secret":"kratos-secret","id":"google","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]},{"client_id":"github-client","client_secret":"kratos-secret","id":"github","issuer_url":"http://127.0.0.1:4444/","mapper_url":"file://test/e2e/profiles/oidc/hydra.jsonnet","provider":"generic","scope":["offline"]}]}`, string(config.Config))
}

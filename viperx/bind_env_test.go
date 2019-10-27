package viperx

import (
	"fmt"
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

	t.Run("func=getSchemaKeys", func(t *testing.T) {
		for k, tc := range []struct {
			schema           string
			expectErr        bool
			expectedKeys     []string
			expectedDefaults map[string]interface{}
		}{
			{
				schema: readFile("./stub/.oathkeeper.schema.json"),

				expectedDefaults: map[string]interface{}{"access_rules.repositories": []string{}, "authenticators.anonymous.config.subject": "anonymous", "authenticators.anonymous.enabled": false, "authenticators.cookie_session.config.only": []string{}, "authenticators.cookie_session.enabled": false, "authenticators.jwt.config.allowed_algorithms": []string{}, "authenticators.jwt.config.jwks_urls": []string{}, "authenticators.jwt.config.required_scope": []string{}, "authenticators.jwt.config.scope_strategy": "none", "authenticators.jwt.config.target_audience": []string{}, "authenticators.jwt.config.trusted_issuers": []string{}, "authenticators.jwt.enabled": false, "authenticators.noop.enabled": false, "authenticators.oauth2_client_credentials.config.required_scope": []string{}, "authenticators.oauth2_client_credentials.enabled": false, "authenticators.oauth2_introspection.config.pre_authorization.enabled": false, "authenticators.oauth2_introspection.config.pre_authorization.scope": []string{}, "authenticators.oauth2_introspection.config.required_scope": []string{}, "authenticators.oauth2_introspection.config.scope_strategy": "none", "authenticators.oauth2_introspection.config.target_audience": []string{}, "authenticators.oauth2_introspection.config.trusted_issuers": []string{}, "authenticators.oauth2_introspection.enabled": false, "authenticators.unauthorized.enabled": false, "authorizers.allow.enabled": false, "authorizers.deny.enabled": false, "authorizers.keto_engine_acp_ory.config.required_action": "unset", "authorizers.keto_engine_acp_ory.config.required_resource": "unset", "authorizers.keto_engine_acp_ory.enabled": false, "log.format": "text", "log.level": "info", "mutators.cookie.enabled": false, "mutators.header.enabled": false, "mutators.hydrator.config.api.retry.delay_in_milliseconds": float64(3), "mutators.hydrator.config.api.retry.number_of_retries": float64(100), "mutators.hydrator.enabled": false, "mutators.id_token.config.ttl": "1m", "mutators.id_token.enabled": false, "mutators.noop.enabled": false, "serve.api.cors.allow_credentials": false, "serve.api.cors.allowed_headers": []string{}, "serve.api.cors.allowed_methods": []string{}, "serve.api.cors.allowed_origins": []string{}, "serve.api.cors.debug": false, "serve.api.cors.enabled": false, "serve.api.cors.exposed_headers": []string{}, "serve.api.cors.max_age": float64(0), "serve.api.host": "", "serve.api.port": float64(4456), "serve.proxy.cors.allow_credentials": false, "serve.proxy.cors.allowed_headers": []string{}, "serve.proxy.cors.allowed_methods": []string{}, "serve.proxy.cors.allowed_origins": []string{}, "serve.proxy.cors.debug": false, "serve.proxy.cors.enabled": false, "serve.proxy.cors.exposed_headers": []string{}, "serve.proxy.cors.max_age": float64(0), "serve.proxy.host": "", "serve.proxy.port": float64(4455), "serve.proxy.timeout.idle": "120s", "serve.proxy.timeout.read": "5s", "serve.proxy.timeout.write": "120s"},

				expectedKeys: []string{
					"serve",
					"serve.api",
					"serve.api.port",
					"serve.api.host",
					"serve.api.cors",

					"serve.api.cors.enabled",
					"serve.api.cors.allowed_origins",
					"serve.api.cors.allowed_methods",

					"serve.api.cors.allowed_headers",
					"serve.api.cors.exposed_headers",
					"serve.api.cors.allow_credentials",

					"serve.api.cors.max_age",
					"serve.api.cors.debug",
					"serve.api.tls",
					"serve.api.tls.key",
					"serve.api.tls.key.path",

					"serve.api.tls.key.base64",
					"serve.api.tls.cert",
					"serve.api.tls.cert.path",
					"serve.api.tls.cert.base64",
					"serve.proxy",

					"serve.proxy.port",
					"serve.proxy.host",
					"serve.proxy.timeout",
					"serve.proxy.timeout.read",
					"serve.proxy.timeout.write",

					"serve.proxy.timeout.idle",
					"serve.proxy.cors",
					"serve.proxy.cors.enabled",
					"serve.proxy.cors.allowed_origins",

					"serve.proxy.cors.allowed_methods",
					"serve.proxy.cors.allowed_headers",
					"serve.proxy.cors.exposed_headers",

					"serve.proxy.cors.allow_credentials",
					"serve.proxy.cors.max_age",
					"serve.proxy.cors.debug",
					"serve.proxy.tls",
					"serve.proxy.tls.key",

					"serve.proxy.tls.key.path",
					"serve.proxy.tls.key.base64",
					"serve.proxy.tls.cert",
					"serve.proxy.tls.cert.path",
					"serve.proxy.tls.cert.base64",

					"access_rules",
					"access_rules.repositories",
					"authenticators",
					"authenticators.anonymous",

					"authenticators.anonymous.enabled",
					"authenticators.anonymous.config",
					"authenticators.anonymous.config.subject",
					"authenticators.noop",

					"authenticators.noop.enabled",
					"authenticators.unauthorized",
					"authenticators.unauthorized.enabled",
					"authenticators.cookie_session",

					"authenticators.cookie_session.enabled",
					"authenticators.cookie_session.config",
					"authenticators.cookie_session.config.check_session_url",

					"authenticators.cookie_session.config.only",
					"authenticators.jwt",
					"authenticators.jwt.enabled",
					"authenticators.jwt.config",

					"authenticators.jwt.config.required_scope",
					"authenticators.jwt.config.target_audience",
					"authenticators.jwt.config.trusted_issuers",

					"authenticators.jwt.config.allowed_algorithms",
					"authenticators.jwt.config.jwks_urls",
					"authenticators.jwt.config.scope_strategy",

					"authenticators.jwt.config.token_from",
					"authenticators.jwt.config.token_from.header",
					"authenticators.jwt.config.token_from.query_parameter",

					"authenticators.oauth2_client_credentials",
					"authenticators.oauth2_client_credentials.enabled",
					"authenticators.oauth2_client_credentials.config",

					"authenticators.oauth2_client_credentials.config.token_url",
					"authenticators.oauth2_client_credentials.config.required_scope",

					"authenticators.oauth2_introspection",
					"authenticators.oauth2_introspection.enabled",
					"authenticators.oauth2_introspection.config",

					"authenticators.oauth2_introspection.config.introspection_url",
					"authenticators.oauth2_introspection.config.scope_strategy",

					"authenticators.oauth2_introspection.config.pre_authorization",
					"authenticators.oauth2_introspection.config.pre_authorization.enabled",

					"authenticators.oauth2_introspection.config.pre_authorization.client_id",
					"authenticators.oauth2_introspection.config.pre_authorization.client_secret",

					"authenticators.oauth2_introspection.config.pre_authorization.token_url",
					"authenticators.oauth2_introspection.config.pre_authorization.scope",

					"authenticators.oauth2_introspection.config.required_scope",
					"authenticators.oauth2_introspection.config.target_audience",

					"authenticators.oauth2_introspection.config.trusted_issuers",
					"authenticators.oauth2_introspection.config.token_from",

					"authenticators.oauth2_introspection.config.token_from.header",
					"authenticators.oauth2_introspection.config.token_from.query_parameter",

					"authorizers",
					"authorizers.allow",
					"authorizers.allow.enabled",
					"authorizers.deny",
					"authorizers.deny.enabled",
					"authorizers.keto_engine_acp_ory",

					"authorizers.keto_engine_acp_ory.enabled",
					"authorizers.keto_engine_acp_ory.config",
					"authorizers.keto_engine_acp_ory.config.base_url",

					"authorizers.keto_engine_acp_ory.config.required_action",
					"authorizers.keto_engine_acp_ory.config.required_resource",
					"authorizers.keto_engine_acp_ory.config.subject",

					"authorizers.keto_engine_acp_ory.config.flavor",
					"mutators",
					"mutators.noop",
					"mutators.noop.enabled",
					"mutators.cookie",

					"mutators.cookie.enabled",
					"mutators.cookie.config",
					"mutators.cookie.config.cookies",
					"mutators.header",

					"mutators.header.enabled",
					"mutators.header.config",
					"mutators.header.config.headers",
					"mutators.hydrator",

					"mutators.hydrator.enabled",
					"mutators.hydrator.config",
					"mutators.hydrator.config.api",
					"mutators.hydrator.config.api.url",

					"mutators.hydrator.config.api.auth",
					"mutators.hydrator.config.api.auth.basic",
					"mutators.hydrator.config.api.auth.basic.username",

					"mutators.hydrator.config.api.auth.basic.password",
					"mutators.hydrator.config.api.retry",
					"mutators.hydrator.config.api.retry.number_of_retries",

					"mutators.hydrator.config.api.retry.delay_in_milliseconds",
					"mutators.id_token",
					"mutators.id_token.enabled",

					"mutators.id_token.config",
					"mutators.id_token.config.claims",
					"mutators.id_token.config.issuer_url",

					"mutators.id_token.config.jwks_url",
					"mutators.id_token.config.ttl",
					"log",
					"log.level",
					"log.format",

					"profiling",
				},
			},
			{
				schema:           readFile("./stub/config.schema.json"),
				expectedKeys:     []string{"dsn"},
				expectedDefaults: map[string]interface{}{},
			},
			{
				schema:    `{"$ref": "http://google/schema.json"}`,
				expectErr: true,
			},
			{
				// this should fail because of recursion
				schema: `{
	"definitions": {
		"foo": {
			"$ref": "#/definitions/foo"
		}
	},
	"type": "object",
	"properties": {
		"bar": {
			"$ref": "#/definitions/foo"
		}
	}
}`,
				expectErr: true,
			},
			{
				schema:           `{"oneOf": [{ "type": "object", "properties": { "list": { "type": "array", "items": { "type": "string" } }, "foo": { "type": "boolean", "default": false}, "bar": { "type": "boolean"} } },{ "type": "object", "properties": { "foo": { "type": "boolean"} } }]}`,
				expectedKeys:     []string{"list", "foo", "bar"},
				expectedDefaults: map[string]interface{}{"foo": false, "list": []string{}},
			},
		} {
			t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
				actualKeys, actualDefaults, err := getSchemaKeys(tc.schema, tc.schema, []string{}, []string{})
				if tc.expectErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.EqualValues(t, tc.expectedKeys, actualKeys)
				assert.EqualValues(t, tc.expectedDefaults, actualDefaults)
			})
		}
	})

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
}

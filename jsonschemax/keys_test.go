package jsonschemax

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readFile(t *testing.T, path string) string {
	schema, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return string(schema)
}

func TestListPaths(t *testing.T) {
	for k, tc := range []struct {
		schema    string
		expectErr bool
		expected  []Path
	}{
		{
			schema: readFile(t, "./stub/.oathkeeper.schema.json"),
			expected: []Path{
				Path{Name: "serve.api.port", Default: float64(4456), Type: float64(0)},
				Path{Name: "serve.api.host", Default: "", Type: ""},
				Path{Name: "serve.api.cors.enabled", Default: false, Type: false},
				Path{Name: "serve.api.cors.allowed_origins", Default: []interface{}{"*"}, Type: []string{}},
				Path{Name: "serve.api.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"}, Type: []string{}},
				Path{Name: "serve.api.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, Type: []string{}},
				Path{Name: "serve.api.cors.exposed_headers", Default: []interface{}{"Content-Type"}, Type: []string{}},
				Path{Name: "serve.api.cors.allow_credentials", Default: false, Type: false},
				Path{Name: "serve.api.cors.max_age", Default: float64(0), Type: float64(0)},
				Path{Name: "serve.api.cors.debug", Default: false, Type: false},
				Path{Name: "serve.api.tls.key.path", Default: nil, Type: ""},
				Path{Name: "serve.api.tls.key.base64", Default: nil, Type: ""},
				Path{Name: "serve.api.tls.cert.path", Default: nil, Type: ""},
				Path{Name: "serve.api.tls.cert.base64", Default: nil, Type: ""},
				Path{Name: "serve.proxy.port", Default: float64(4455), Type: float64(0)},
				Path{Name: "serve.proxy.host", Default: "", Type: ""},
				Path{Name: "serve.proxy.timeout.read", Default: "5s", Type: ""},
				Path{Name: "serve.proxy.timeout.write", Default: "120s", Type: ""},
				Path{Name: "serve.proxy.timeout.idle", Default: "120s", Type: ""},
				Path{Name: "serve.proxy.cors.enabled", Default: false, Type: false},
				Path{Name: "serve.proxy.cors.allowed_origins", Default: []interface{}{"*"}, Type: []string{}},
				Path{Name: "serve.proxy.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"}, Type: []string{}},
				Path{Name: "serve.proxy.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, Type: []string{}},
				Path{Name: "serve.proxy.cors.exposed_headers", Default: []interface{}{"Content-Type"}, Type: []string{}},
				Path{Name: "serve.proxy.cors.allow_credentials", Default: false, Type: false},
				Path{Name: "serve.proxy.cors.max_age", Default: float64(0), Type: float64(0)},
				Path{Name: "serve.proxy.cors.debug", Default: false, Type: false},
				Path{Name: "serve.proxy.tls.key.path", Default: nil, Type: ""},
				Path{Name: "serve.proxy.tls.key.base64", Default: nil, Type: ""},
				Path{Name: "serve.proxy.tls.cert.path", Default: nil, Type: ""},
				Path{Name: "serve.proxy.tls.cert.base64", Default: nil, Type: ""},
				Path{Name: "access_rules.repositories", Default: nil, Type: []string{}},
				Path{Name: "authenticators.anonymous.enabled", Default: false, Type: false},
				Path{Name: "authenticators.anonymous.config.subject", Default: "anonymous", Type: ""},
				Path{Name: "authenticators.noop.enabled", Default: false, Type: false},
				Path{Name: "authenticators.unauthorized.enabled", Default: false, Type: false},
				Path{Name: "authenticators.cookie_session.enabled", Default: false, Type: false},
				Path{Name: "authenticators.cookie_session.config.check_session_url", Default: nil, Type: ""},
				Path{Name: "authenticators.cookie_session.config.only", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.enabled", Default: false, Type: false},
				Path{Name: "authenticators.jwt.config.required_scope", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.config.target_audience", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.config.trusted_issuers", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.config.allowed_algorithms", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.config.jwks_urls", Default: nil, Type: []string{}},
				Path{Name: "authenticators.jwt.config.scope_strategy", Default: "none", Type: ""},
				Path{Name: "authenticators.jwt.config.token_from.header", Default: nil, Type: ""},
				Path{Name: "authenticators.jwt.config.token_from.query_parameter", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_client_credentials.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_client_credentials.config.token_url", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_client_credentials.config.required_scope", Default: nil, Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_introspection.config.introspection_url", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.scope_strategy", Default: "none", Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.client_id", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.client_secret", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.token_url", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.scope", Default: nil, Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.required_scope", Default: nil, Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.target_audience", Default: nil, Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.trusted_issuers", Default: nil, Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.token_from.header", Default: nil, Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.token_from.query_parameter", Default: nil, Type: ""},
				Path{Name: "authorizers.allow.enabled", Default: false, Type: false},
				Path{Name: "authorizers.deny.enabled", Default: false, Type: false},
				Path{Name: "authorizers.keto_engine_acp_ory.enabled", Default: false, Type: false},
				Path{Name: "authorizers.keto_engine_acp_ory.config.base_url", Default: nil, Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.required_action", Default: "unset", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.required_resource", Default: "unset", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.subject", Default: nil, Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.flavor", Default: nil, Type: ""},
				Path{Name: "mutators.noop.enabled", Default: false, Type: false},
				Path{Name: "mutators.cookie.enabled", Default: false, Type: false},
				Path{Name: "mutators.cookie.config.cookies", Default: nil, Type: map[string]interface{}{}},
				Path{Name: "mutators.header.enabled", Default: false, Type: false},
				Path{Name: "mutators.header.config.headers", Default: nil, Type: map[string]interface{}{}},
				Path{Name: "mutators.hydrator.enabled", Default: false, Type: false},
				Path{Name: "mutators.hydrator.config.api.url", Default: nil, Type: ""},
				Path{Name: "mutators.hydrator.config.api.auth.basic.username", Default: nil, Type: ""},
				Path{Name: "mutators.hydrator.config.api.auth.basic.password", Default: nil, Type: ""},
				Path{Name: "mutators.hydrator.config.api.retry.number_of_retries", Default: float64(100), Type: float64(0)},
				Path{Name: "mutators.hydrator.config.api.retry.delay_in_milliseconds", Default: float64(3), Type: float64(0)},
				Path{Name: "mutators.id_token.enabled", Default: false, Type: false},
				Path{Name: "mutators.id_token.config.claims", Default: nil, Type: ""},
				Path{Name: "mutators.id_token.config.issuer_url", Default: nil, Type: ""},
				Path{Name: "mutators.id_token.config.jwks_url", Default: nil, Type: ""},
				Path{Name: "mutators.id_token.config.ttl", Default: "1m", Type: ""},
				Path{Name: "log.level", Default: "info", Type: ""},
				Path{Name: "log.format", Default: "text", Type: ""},
				Path{Name: "profiling", Default: nil, Type: ""},
			},
		},
		{
			schema: readFile(t, "./stub/config.schema.json"),
			expected: []Path{
				{
					Name:    "dsn",
					Default: nil,
					Type:    "",
				},
			},
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
			schema:    `{"$ref": "http://google/schema.json"}`,
			expectErr: true,
		},
		{
			schema: `{"oneOf": [{ "type": "object", "properties": { "list": { "type": "array", "items": { "type": "string" } }, "foo": { "type": "boolean", "default": false}, "bar": { "type": "boolean"} } },{ "type": "object", "properties": { "foo": { "type": "boolean"} } }]}`,
			expected: []Path{
				{
					Name: "list",
					Type: []string{},
				},
				{
					Name:    "foo",
					Default: false,
					Type:    false,
				},
				{
					Name: "bar",
					Type: false,
				},
				{
					Name: "foo",
					Type: false,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			actual, err := ListPaths(tc.schema)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.EqualValues(t, tc.expected, actual)
		})
	}
}

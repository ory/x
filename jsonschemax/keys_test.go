package jsonschemax

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"regexp"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
)

const recursiveSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "test.json",
  "definitions": {
    "foo": {
      "type": "object",
      "properties": {
		"bars": {
			"type": "string",
			"format": "email",
			"pattern": ".*"
		},
        "bar": {
          "$ref": "#/definitions/bar"
        }
      }
    },
    "bar": {
      "type": "object",
      "properties": {
		"foos": {
		  "type": "string",
		  "minLength": 1,
		  "maxLength": 10
		},
        "foo": {
          "$ref": "#/definitions/foo"
        }
      }
    }
  },
  "type": "object",
  "properties": {
    "bar": {
      "$ref": "#/definitions/bar"
    }
  }
}`

func readFile(t *testing.T, path string) string {
	schema, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return string(schema)
}

func assertEqualPaths(t *testing.T, expected byName, actual byName) {
	for i := range expected {
		t.Run("path="+expected[i].Name, func(t *testing.T) {
			e := expected[i]
			// because default if not given is -1
			if e.MinLength == 0 {
				e.MinLength = -1
			}
			if e.MaxLength == 0 {
				e.MaxLength = -1
			}

			a := actual[i]
			assert.Equal(t, e.Pattern, a.Pattern, fmt.Sprintf("path: %s\n", e.Name))

			e.Pattern = nil
			a.Pattern = nil

			if e.Minimum != nil {
				assert.NotNil(t, a.Minimum)
				assert.Equal(t, e.Minimum.String(), e.Minimum.String(), fmt.Sprintf("path: %s\n", e.Name))
			} else {
				assert.Nil(t, a.Minimum)
			}
			if e.Maximum != nil {
				assert.NotNil(t, a.Maximum)
				assert.Equal(t, e.Maximum.String(), e.Maximum.String(), fmt.Sprintf("path: %s\n", e.Name))
			} else {
				assert.Nil(t, a.Maximum)
			}

			e.Minimum = nil
			a.Minimum = nil
			e.Maximum = nil
			a.Maximum = nil

			assert.Equal(t, e, a)
		})
	}
}

const fooExtensionName = "fooExtension"

type (
	extensionConfig struct {
		NotAJSONSchemaKey string `json:"not-a-json-schema-key"`
	}
)

func fooExtensionCompile(_ jsonschema.CompilerContext, m map[string]interface{}) (interface{}, error) {
	if raw, ok := m[fooExtensionName]; ok {
		var b bytes.Buffer
		if err := json.NewEncoder(&b).Encode(raw); err != nil {
			return nil, errors.WithStack(err)
		}

		var e extensionConfig
		if err := json.NewDecoder(&b).Decode(&e); err != nil {
			return nil, errors.WithStack(err)
		}

		return &e, nil
	}
	return nil, nil
}

func fooExtensionValidate(_ jsonschema.ValidationContext, _, _ interface{}) error {
	return nil
}

func (ec *extensionConfig) EnhancePath(p Path) map[string]interface{} {
	if ec.NotAJSONSchemaKey != "" {
		fmt.Printf("enhancing path: %s with custom property %s\n", p.Name, ec.NotAJSONSchemaKey)
		return map[string]interface{}{
			ec.NotAJSONSchemaKey: p.Name,
		}
	}
	return nil
}

func TestListPathsWithRecursion(t *testing.T) {
	for k, tc := range []struct {
		recursion uint8
		expected  byName
	}{
		{
			recursion: 5,
			expected: byName{

				Path{
					Name:      "bar.foo.bar.foo.bar.foos",
					Default:   interface{}(nil),
					Type:      "",
					TypeHint:  String,
					MaxLength: 10,
					MinLength: 1,
				},

				Path{
					Name:      "bar.foo.bar.foo.bars",
					Default:   interface{}(nil),
					Type:      "",
					Format:    "email",
					TypeHint:  String,
					Pattern:   regexp.MustCompile(".*"),
					MaxLength: -1,
					MinLength: -1,
				},

				Path{
					Name:      "bar.foo.bar.foos",
					Default:   interface{}(nil),
					Type:      "",
					TypeHint:  String,
					MaxLength: 10,
					MinLength: 1,
				},

				Path{
					Name:      "bar.foo.bars",
					Default:   interface{}(nil),
					Type:      "",
					TypeHint:  String,
					Format:    "email",
					Pattern:   regexp.MustCompile(".*"),
					MaxLength: -1,
					MinLength: -1,
				},

				Path{
					Name:      "bar.foos",
					Default:   interface{}(nil),
					Type:      "",
					TypeHint:  String,
					MaxLength: 10,
					MinLength: 1,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			c := jsonschema.NewCompiler()
			require.NoError(t, c.AddResource("test.json", bytes.NewBufferString(recursiveSchema)))
			actual, err := ListPathsWithRecursion("test.json", c, tc.recursion)
			require.NoError(t, err)

			assertEqualPaths(t, tc.expected, actual)
		})
	}
}

func TestListPaths(t *testing.T) {
	for k, tc := range []struct {
		schema    string
		expectErr bool
		expected  byName
		extension *jsonschema.Extension
	}{
		{
			schema: readFile(t, "./stub/.oathkeeper.schema.json"),
			expected: byName{
				Path{Title: "Repositories", Description: "Locations (list of URLs) where access rules should be fetched from on boot. It is expected that the documents at those locations return a JSON or YAML Array containing ORY Oathkeeper Access Rules:\n\n- If the URL Scheme is `file://`, the access rules (an array of access rules is expected) will be fetched from the local file system.\n- If the URL Scheme is `inline://`, the access rules (an array of access rules is expected) are expected to be a base64 encoded (with padding!) JSON/YAML string (base64_encode(`[{\"id\":\"foo-rule\",\"authenticators\":[....]}]`)).\n- If the URL Scheme is `http://` or `https://`, the access rules (an array of access rules is expected) will be fetched from the provided HTTP(s) location.", Examples: []interface{}{"[\"file://path/to/rules.json\",\"inline://W3siaWQiOiJmb28tcnVsZSIsImF1dGhlbnRpY2F0b3JzIjpbXX1d\",\"https://path-to-my-rules/rules.json\"]"}, Name: "access_rules.repositories", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Anonymous Subject", Description: "Sets the anonymous username.", Examples: []interface{}{"guest", "anon", "anonymous", "unknown"}, Name: "authenticators.anonymous.config.subject", Default: "anonymous", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.anonymous.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Session Check URL", Description: "The origin to proxy requests to. If the response is a 200 with body `{ \"subject\": \"...\", \"extra\": {} }`. The request will pass the subject through successfully, otherwise it will be marked as unauthorized.\n\n>If this authenticator is enabled, this value is required.", Examples: []interface{}{"https://session-store-host"}, Name: "authenticators.cookie_session.config.check_session_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Only Cookies", Description: "A list of possible cookies to look for on incoming requests, and will fallthrough to the next authenticator if none of the passed cookies are set on the request.", Name: "authenticators.cookie_session.config.only", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.cookie_session.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authenticators.jwt.config.allowed_algorithms", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "JSON Web Key URLs", Description: "URLs where ORY Oathkeeper can retrieve JSON Web Keys from for validating the JSON Web Token. Usually something like \"https://my-keys.com/.well-known/jwks.json\". The response of that endpoint must return a JSON Web Key Set (JWKS).\n\n>If this authenticator is enabled, this value is required.", Examples: []interface{}{"https://my-website.com/.well-known/jwks.json", "https://my-other-website.com/.well-known/jwks.json", "file://path/to/local/jwks.json"}, Name: "authenticators.jwt.config.jwks_urls", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Required Token Scope", Description: "An array of OAuth 2.0 scopes that are required when accessing an endpoint protected by this handler.\n If the token used in the Authorization header did not request that specific scope, the request is denied.", Name: "authenticators.jwt.config.required_scope", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Scope Strategy", Description: "Sets the strategy validation algorithm.", Name: "authenticators.jwt.config.scope_strategy", Default: "none", Type: "", TypeHint: String, Format: "", Enum: []interface{}{"hierarchic", "exact", "wildcard", "none"}, ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Intended Audience", Description: "An array of audiences that are required when accessing an endpoint protected by this handler.\n If the token used in the Authorization header is not intended for any of the requested audiences, the request is denied.", Name: "authenticators.jwt.config.target_audience", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Header", Description: "The header (case insensitive) that must contain a token for request authentication. It can't be set along with query_parameter.", Name: "authenticators.jwt.config.token_from.header", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Query Parameter", Description: "The query parameter (case sensitive) that must contain a token for request authentication. It can't be set along with header.", Name: "authenticators.jwt.config.token_from.query_parameter", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authenticators.jwt.config.trusted_issuers", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.jwt.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.noop.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Request Permissions (Token Scope)", Description: "Scopes is an array of OAuth 2.0 scopes that are required when accessing an endpoint protected by this rule.\n If the token used in the Authorization header did not request that specific scope, the request is denied.", Name: "authenticators.oauth2_client_credentials.config.required_scope", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "The OAuth 2.0 Token Endpoint that will be used to validate the client credentials.\n\n>If this authenticator is enabled, this value is required.", Examples: []interface{}{"https://my-website.com/oauth2/token"}, Name: "authenticators.oauth2_client_credentials.config.token_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.oauth2_client_credentials.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "OAuth 2.0 Introspection URL", Description: "The OAuth 2.0 Token Introspection endpoint URL.\n\n>If this authenticator is enabled, this value is required.", Examples: []interface{}{"https://my-website.com/oauth2/introspection"}, Name: "authenticators.oauth2_introspection.config.introspection_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "OAuth 2.0 Client ID", Description: "The OAuth 2.0 Client ID to be used for the OAuth 2.0 Client Credentials Grant.\n\n>If pre-authorization is enabled, this value is required.", Name: "authenticators.oauth2_introspection.config.pre_authorization.client_id", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "OAuth 2.0 Client Secret", Description: "The OAuth 2.0 Client Secret to be used for the OAuth 2.0 Client Credentials Grant.\n\n>If pre-authorization is enabled, this value is required.", Name: "authenticators.oauth2_introspection.config.pre_authorization.client_secret", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "", Name: "authenticators.oauth2_introspection.config.pre_authorization.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "OAuth 2.0 Scope", Description: "The OAuth 2.0 Scope to be requested during the OAuth 2.0 Client Credentials Grant.", Examples: []interface{}{[]interface{}{"[\"foo\", \"bar\"]"}}, Name: "authenticators.oauth2_introspection.config.pre_authorization.scope", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "OAuth 2.0 Token URL", Description: "The OAuth 2.0 Token Endpoint where the OAuth 2.0 Client Credentials Grant will be performed.\n\n>If pre-authorization is enabled, this value is required.", Name: "authenticators.oauth2_introspection.config.pre_authorization.token_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Required Scope", Description: "An array of OAuth 2.0 scopes that are required when accessing an endpoint protected by this handler.\n If the token used in the Authorization header did not request that specific scope, the request is denied.", Name: "authenticators.oauth2_introspection.config.required_scope", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Scope Strategy", Description: "Sets the strategy validation algorithm.", Name: "authenticators.oauth2_introspection.config.scope_strategy", Default: "none", Type: "", TypeHint: String, Format: "", Enum: []interface{}{"hierarchic", "exact", "wildcard", "none"}, ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Target Audience", Description: "An array of audiences that are required when accessing an endpoint protected by this handler.\n If the token used in the Authorization header is not intended for any of the requested audiences, the request is denied.", Name: "authenticators.oauth2_introspection.config.target_audience", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Token From", Description: "The location of the token.\n If not configured, the token will be received from a default location - 'Authorization' header.\n One and only one location (header or query) must be specified.", Name: "authenticators.oauth2_introspection.config.token_from", Type: map[string]interface{}{}, TypeHint: JSON, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Header", Description: "The header (case insensitive) that must contain a token for request authentication.\n It can't be set along with query_parameter.", Name: "authenticators.oauth2_introspection.config.token_from.header", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Query Parameter", Description: "The query parameter (case sensitive) that must contain a token for request authentication.\n It can't be set along with header.", Name: "authenticators.oauth2_introspection.config.token_from.query_parameter", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Trusted Issuers", Description: "The token must have been issued by one of the issuers listed in this array.", Name: "authenticators.oauth2_introspection.config.trusted_issuers", Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.oauth2_introspection.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authenticators.unauthorized.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authorizers.allow.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authorizers.deny.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Base URL", Description: "The base URL of ORY Keto.\n\n>If this authorizer is enabled, this value is required.", Examples: []interface{}{"http://my-keto/"}, Name: "authorizers.keto_engine_acp_ory.config.base_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authorizers.keto_engine_acp_ory.config.flavor", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authorizers.keto_engine_acp_ory.config.required_action", Default: "unset", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authorizers.keto_engine_acp_ory.config.required_resource", Default: "unset", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "authorizers.keto_engine_acp_ory.config.subject", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "authorizers.keto_engine_acp_ory.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Format", Description: "The log format can either be text or JSON.", Name: "log.format", Default: "text", Type: "", TypeHint: String, Format: "", Enum: []interface{}{"text", "json"}, ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Level", Description: "Debug enables stack traces on errors. Can also be set using environment variable LOG_LEVEL.", Name: "log.level", Default: "info", Type: "", TypeHint: String, Format: "", Enum: []interface{}{"panic", "fatal", "error", "warn", "info", "debug"}, ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.cookie.config.cookies", Type: map[string]interface{}{}, TypeHint: JSON, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "mutators.cookie.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.header.config.headers", Type: map[string]interface{}{}, TypeHint: JSON, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "mutators.header.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.hydrator.config.api.auth.basic.password", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.hydrator.config.api.auth.basic.username", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.hydrator.config.api.retry.delay_in_milliseconds", Default: float64(3), Type: float64(0), TypeHint: Int, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1, Minimum: big.NewFloat(0)},
				Path{Title: "", Description: "", Name: "mutators.hydrator.config.api.retry.number_of_retries", Default: float64(100), Type: float64(0), TypeHint: Float, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1, Minimum: big.NewFloat(0)},
				Path{Title: "", Description: "", Name: "mutators.hydrator.config.api.url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "mutators.hydrator.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "", Description: "", Name: "mutators.id_token.config.claims", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Issuer URL", Description: "Sets the \"iss\" value of the ID Token.\n\n>If this mutator is enabled, this value is required.", Name: "mutators.id_token.config.issuer_url", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "JSON Web Key URL", Description: "Sets the URL where keys should be fetched from. Supports remote locations (http, https) as well as local filesystem paths.\n\n>If this mutator is enabled, this value is required.", Examples: []interface{}{"https://fetch-keys/from/this/location.json", "file:///from/this/absolute/location.json", "file://../from/this/relative/location.json"}, Name: "mutators.id_token.config.jwks_url", Type: "", TypeHint: String, Format: "uri", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Expire After", Description: "Sets the time-to-live of the JSON Web Token.", Examples: []interface{}{"1h", "1m", "30s"}, Name: "mutators.id_token.config.ttl", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$"), Default: "1m", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "mutators.id_token.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enabled", Description: "En-/disables this component.", Examples: []interface{}{true}, Name: "mutators.noop.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Profiling", Description: "Enables CPU or memory profiling if set. For more details on profiling Go programs read [Profiling Go Programs](https://blog.golang.org/profiling-go-programs).", Name: "profiling", Type: "", TypeHint: String, Format: "", Enum: []interface{}{"cpu", "mem"}, ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allow HTTP Credentials", Description: "Indicates whether the request can include user credentials like cookies, HTTP authentication or client side SSL certificates.", Name: "serve.api.cors.allow_credentials", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Request HTTP Headers", Description: "A list of non simple headers the client is allowed to use with cross-domain requests.", Name: "serve.api.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: 1, MaxLength: -1},
				Path{Title: "Allowed HTTP Methods", Description: "A list of methods the client is allowed to use with cross-domain requests.", Name: "serve.api.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Origins", Description: "A list of origins a cross-domain request can be executed from. If the special * value is present in the list, all origins will be allowed. An origin may contain a wildcard (*) to replace 0 or more characters (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penality. Only one wildcard can be used per origin.", Examples: []interface{}{"https://example.com", "https://*.example.com", "https://*.foo.example.com"}, Name: "serve.api.cors.allowed_origins", Default: []interface{}{"*"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enable Debugging", Description: "Set to true to debug server side CORS issues.", Name: "serve.api.cors.debug", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enable CORS", Description: "If set to true, CORS will be enabled and preflight-requests (OPTION) will be answered.", Name: "serve.api.cors.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Response HTTP Headers", Description: "Indicates which headers are safe to expose to the API of a CORS API specification", Name: "serve.api.cors.exposed_headers", Default: []interface{}{"Content-Type"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: 1, MaxLength: -1},
				Path{Title: "Maximum Age", Description: "Indicates how long (in seconds) the results of a preflight request can be cached. The default is 0 which stands for no max age.", Name: "serve.api.cors.max_age", Default: float64(0), Type: float64(0), TypeHint: Float, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Host", Description: "The network interface to listen on.", Examples: []interface{}{"localhost", "127.0.0.1"}, Name: "serve.api.host", Default: "", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Port", Description: "The port to listen on.", Name: "serve.api.port", Default: float64(4456), Type: float64(0), TypeHint: Float, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Base64 Encoded Inline", Description: "The base64 string of the PEM-encoded file content. Can be generated using for example `base64 -i path/to/file.pem`.", Examples: []interface{}{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tXG5NSUlEWlRDQ0FrMmdBd0lCQWdJRVY1eE90REFOQmdr..."}, Name: "serve.api.tls.cert.base64", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Path to PEM-encoded Fle", Description: "", Examples: []interface{}{"path/to/file.pem"}, Name: "serve.api.tls.cert.path", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Base64 Encoded Inline", Description: "The base64 string of the PEM-encoded file content. Can be generated using for example `base64 -i path/to/file.pem`.", Examples: []interface{}{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tXG5NSUlEWlRDQ0FrMmdBd0lCQWdJRVY1eE90REFOQmdr..."}, Name: "serve.api.tls.key.base64", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Path to PEM-encoded Fle", Description: "", Examples: []interface{}{"path/to/file.pem"}, Name: "serve.api.tls.key.path", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allow HTTP Credentials", Description: "Indicates whether the request can include user credentials like cookies, HTTP authentication or client side SSL certificates.", Name: "serve.proxy.cors.allow_credentials", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Request HTTP Headers", Description: "A list of non simple headers the client is allowed to use with cross-domain requests.", Name: "serve.proxy.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: 1, MaxLength: -1},
				Path{Title: "Allowed HTTP Methods", Description: "A list of methods the client is allowed to use with cross-domain requests.", Name: "serve.proxy.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Origins", Description: "A list of origins a cross-domain request can be executed from. If the special * value is present in the list, all origins will be allowed. An origin may contain a wildcard (*) to replace 0 or more characters (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penality. Only one wildcard can be used per origin.", Examples: []interface{}{"https://example.com", "https://*.example.com", "https://*.foo.example.com"}, Name: "serve.proxy.cors.allowed_origins", Default: []interface{}{"*"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enable Debugging", Description: "Set to true to debug server side CORS issues.", Name: "serve.proxy.cors.debug", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Enable CORS", Description: "If set to true, CORS will be enabled and preflight-requests (OPTION) will be answered.", Name: "serve.proxy.cors.enabled", Default: false, Type: false, TypeHint: Bool, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Allowed Response HTTP Headers", Description: "Indicates which headers are safe to expose to the API of a CORS API specification", Name: "serve.proxy.cors.exposed_headers", Default: []interface{}{"Content-Type"}, Type: []string{}, TypeHint: StringSlice, Format: "", ReadOnly: false, MinLength: 1, MaxLength: -1},
				Path{Title: "Maximum Age", Description: "Indicates how long (in seconds) the results of a preflight request can be cached. The default is 0 which stands for no max age.", Name: "serve.proxy.cors.max_age", Default: float64(0), Type: float64(0), TypeHint: Float, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Host", Description: "The network interface to listen on. Leave empty to listen on all interfaces.", Examples: []interface{}{"localhost", "127.0.0.1"}, Name: "serve.proxy.host", Default: "", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Port", Description: "The port to listen on.", Name: "serve.proxy.port", Default: float64(4455), Type: float64(0), TypeHint: Float, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "HTTP Idle Timeout", Description: " The maximum amount of time to wait for any action of a request session, reading data or writing the response.", Examples: []interface{}{"5s", "5m", "5h"}, Name: "serve.proxy.timeout.idle", Default: "120s", Type: "", TypeHint: String, Format: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$"), ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "HTTP Read Timeout", Description: "The maximum duration for reading the entire request, including the body.", Examples: []interface{}{"5s", "5m", "5h"}, Name: "serve.proxy.timeout.read", Default: "5s", Type: "", TypeHint: String, Format: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$"), ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "HTTP Write Timeout", Description: "The maximum duration before timing out writes of the response. Increase this parameter to prevent unexpected closing a client connection if an upstream request is responding slowly.", Examples: []interface{}{"5s", "5m", "5h"}, Name: "serve.proxy.timeout.write", Default: "120s", Type: "", TypeHint: String, Format: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$"), ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Base64 Encoded Inline", Description: "The base64 string of the PEM-encoded file content. Can be generated using for example `base64 -i path/to/file.pem`.", Examples: []interface{}{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tXG5NSUlEWlRDQ0FrMmdBd0lCQWdJRVY1eE90REFOQmdr..."}, Name: "serve.proxy.tls.cert.base64", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Path to PEM-encoded Fle", Description: "", Examples: []interface{}{"path/to/file.pem"}, Name: "serve.proxy.tls.cert.path", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Base64 Encoded Inline", Description: "The base64 string of the PEM-encoded file content. Can be generated using for example `base64 -i path/to/file.pem`.", Examples: []interface{}{"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tXG5NSUlEWlRDQ0FrMmdBd0lCQWdJRVY1eE90REFOQmdr..."}, Name: "serve.proxy.tls.key.base64", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
				Path{Title: "Path to PEM-encoded Fle", Description: "", Examples: []interface{}{"path/to/file.pem"}, Name: "serve.proxy.tls.key.path", Type: "", TypeHint: String, Format: "", ReadOnly: false, MinLength: -1, MaxLength: -1},
			},
		},
		{
			schema: readFile(t, "./stub/config.schema.json"),
			expected: []Path{
				{
					Name:     "dsn",
					Default:  nil,
					TypeHint: String,
					Type:     "",
				},
			},
		},
		{
			// this should fail because of recursion
			schema:    recursiveSchema,
			expectErr: true,
		},
		{
			schema: `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "test.json",
  "oneOf": [
    {
      "type": "object",
      "properties": {
        "list": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "foo": {
          "default": false,
          "type": "boolean"
        },
        "bar": {
          "type": "boolean",
          "default": "asdf",
          "readOnly": true
        }
      }
    },
    {
      "type": "object",
      "properties": {
        "foo": {
          "type": "boolean"
        }
      }
    }
  ]
}`,
			expected: byName{
				{
					Name:     "bar",
					Default:  "asdf",
					Type:     false,
					TypeHint: Bool,
					ReadOnly: true,
				},
				{
					Name:     "foo",
					Default:  false,
					Type:     false,
					TypeHint: Bool,
				},
				{
					Name:     "list",
					Type:     []string{},
					TypeHint: StringSlice,
				},
			},
		},
		{
			schema: `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "test.json",
  "type": "object",
  "properties": {
    "foo": {
      "type": "boolean"
    },
    "bar": {
      "type": "string",
      "fooExtension": {
        "not-a-json-schema-key": "foobar"
      }
    }
  }
}`,
			extension: &jsonschema.Extension{
				Meta:     nil,
				Compile:  fooExtensionCompile,
				Validate: fooExtensionValidate,
			},
			expected: byName{
				{
					Name:     "bar",
					Type:     "",
					TypeHint: String,
					CustomProperties: map[string]interface{}{
						"foobar": "bar",
					},
				},
				{
					Name:     "foo",
					Type:     false,
					TypeHint: Bool,
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			c := jsonschema.NewCompiler()
			if tc.extension != nil {
				c.Extensions[fooExtensionName] = *tc.extension
			}

			require.NoError(t, c.AddResource("test.json", bytes.NewBufferString(tc.schema)))
			actual, err := ListPaths("test.json", c)
			if tc.expectErr {
				require.Error(t, err, "%+v", actual)
				return
			}
			require.NoError(t, err)
			assertEqualPaths(t, tc.expected, actual)
		})
	}
}

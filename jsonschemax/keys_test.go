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
					MaxLength: 10,
					MinLength: 1,
				},
				Path{
					Name:      "bar.foo.bar.foo.bars",
					Default:   interface{}(nil),
					Type:      "",
					Format:    "email",
					Pattern:   regexp.MustCompile(".*"),
					MaxLength: -1,
					MinLength: -1,
				},
				Path{
					Name:      "bar.foo.bar.foos",
					Default:   interface{}(nil),
					Type:      "",
					MaxLength: 10,
					MinLength: 1,
				},
				Path{
					Name:      "bar.foo.bars",
					Default:   interface{}(nil),
					Type:      "",
					Format:    "email",
					Pattern:   regexp.MustCompile(".*"),
					MaxLength: -1,
					MinLength: -1,
				},
				Path{
					Name:      "bar.foos",
					Default:   interface{}(nil),
					Type:      "",
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
				Path{Name: "access_rules.repositories", Type: []string{}},
				Path{Name: "authenticators.anonymous.config.subject", Default: "anonymous", Type: ""},
				Path{Name: "authenticators.anonymous.enabled", Default: false, Type: false},
				Path{Name: "authenticators.cookie_session.config.check_session_url", Type: "", Format: "uri"},
				Path{Name: "authenticators.cookie_session.config.only", Type: []string{}},
				Path{Name: "authenticators.cookie_session.enabled", Default: false, Type: false},
				Path{Name: "authenticators.jwt.config.allowed_algorithms", Type: []string{}},
				Path{Name: "authenticators.jwt.config.jwks_urls", Type: []string{}},
				Path{Name: "authenticators.jwt.config.required_scope", Type: []string{}},
				Path{Name: "authenticators.jwt.config.scope_strategy", Default: "none", Type: "", Enum: []interface{}{"hierarchic", "exact", "wildcard", "none"}},
				Path{Name: "authenticators.jwt.config.target_audience", Type: []string{}},
				Path{Name: "authenticators.jwt.config.token_from.header", Type: ""},
				Path{Name: "authenticators.jwt.config.token_from.query_parameter", Type: ""},
				Path{Name: "authenticators.jwt.config.trusted_issuers", Type: []string{}},
				Path{Name: "authenticators.jwt.enabled", Default: false, Type: false},
				Path{Name: "authenticators.noop.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_client_credentials.config.required_scope", Type: []string{}},
				Path{Name: "authenticators.oauth2_client_credentials.config.token_url", Type: "", Format: "uri"},
				Path{Name: "authenticators.oauth2_client_credentials.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_introspection.config.introspection_url", Type: "", Format: "uri"},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.client_id", Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.client_secret", Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.enabled", Default: false, Type: false},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.scope", Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.pre_authorization.token_url", Type: "", Format: "uri"},
				Path{Name: "authenticators.oauth2_introspection.config.required_scope", Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.scope_strategy", Default: "none", Type: "", Enum: []interface{}{"hierarchic", "exact", "wildcard", "none"}},
				Path{Name: "authenticators.oauth2_introspection.config.target_audience", Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.config.token_from", Type: map[string]interface{}{}},
				Path{Name: "authenticators.oauth2_introspection.config.token_from.header", Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.token_from.query_parameter", Type: ""},
				Path{Name: "authenticators.oauth2_introspection.config.trusted_issuers", Type: []string{}},
				Path{Name: "authenticators.oauth2_introspection.enabled", Default: false, Type: false},
				Path{Name: "authenticators.unauthorized.enabled", Default: false, Type: false},
				Path{Name: "authorizers.allow.enabled", Default: false, Type: false},
				Path{Name: "authorizers.deny.enabled", Default: false, Type: false},
				Path{Name: "authorizers.keto_engine_acp_ory.config.base_url", Type: "", Format: "uri"},
				Path{Name: "authorizers.keto_engine_acp_ory.config.flavor", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.required_action", Default: "unset", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.required_resource", Default: "unset", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.config.subject", Type: ""},
				Path{Name: "authorizers.keto_engine_acp_ory.enabled", Default: false, Type: false},
				Path{Name: "log.format", Default: "text", Type: "", Enum: []interface{}{"text", "json"}},
				Path{Name: "log.level", Default: "info", Type: "", Enum: []interface{}{"panic", "fatal", "error", "warn", "info", "debug"}},
				Path{Name: "mutators.cookie.config.cookies", Type: map[string]interface{}{}},
				Path{Name: "mutators.cookie.enabled", Default: false, Type: false},
				Path{Name: "mutators.header.config.headers", Type: map[string]interface{}{}},
				Path{Name: "mutators.header.enabled", Default: false, Type: false},
				Path{Name: "mutators.hydrator.config.api.auth.basic.password", Type: ""},
				Path{Name: "mutators.hydrator.config.api.auth.basic.username", Type: ""},
				Path{Name: "mutators.hydrator.config.api.retry.delay_in_milliseconds", Default: float64(3), Type: float64(0), Minimum: big.NewFloat(0)},
				Path{Name: "mutators.hydrator.config.api.retry.number_of_retries", Default: float64(100), Type: float64(0), Minimum: big.NewFloat(0)},
				Path{Name: "mutators.hydrator.config.api.url", Type: "", Format: "uri"},
				Path{Name: "mutators.hydrator.enabled", Default: false, Type: false},
				Path{Name: "mutators.id_token.config.claims", Type: ""},
				Path{Name: "mutators.id_token.config.issuer_url", Type: ""},
				Path{Name: "mutators.id_token.config.jwks_url", Type: "", Format: "uri"},
				Path{Name: "mutators.id_token.config.ttl", Default: "1m", Type: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$")},
				Path{Name: "mutators.id_token.enabled", Default: false, Type: false},
				Path{Name: "mutators.noop.enabled", Default: false, Type: false},
				Path{Name: "profiling", Type: "", Enum: []interface{}{"cpu", "mem"}},
				Path{Name: "serve.api.cors.allow_credentials", Default: false, Type: false},
				Path{Name: "serve.api.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, MinLength: 1,
					Type: []string{}},
				Path{Name: "serve.api.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"},
					Type: []string{}},
				Path{Name: "serve.api.cors.allowed_origins", Default: []interface{}{"*"},
					Type: []string{}},
				Path{Name: "serve.api.cors.debug", Default: false, Type: false},
				Path{Name: "serve.api.cors.enabled", Default: false, Type: false},
				Path{Name: "serve.api.cors.exposed_headers", Default: []interface{}{"Content-Type"}, MinLength: 1,
					Type: []string{}},
				Path{Name: "serve.api.cors.max_age", Default: float64(0), Type: float64(0)},
				Path{Name: "serve.api.host", Default: "", Type: ""},
				Path{Name: "serve.api.port", Default: float64(4456), Type: float64(0)},
				Path{Name: "serve.api.tls.cert.base64", Type: ""},
				Path{Name: "serve.api.tls.cert.path", Type: ""},
				Path{Name: "serve.api.tls.key.base64", Type: ""},
				Path{Name: "serve.api.tls.key.path", Type: ""},
				Path{Name: "serve.proxy.cors.allow_credentials", Default: false, Type: false},
				Path{Name: "serve.proxy.cors.allowed_headers", Default: []interface{}{"Authorization", "Content-Type"}, MinLength: 1,
					Type: []string{}},
				Path{Name: "serve.proxy.cors.allowed_methods", Default: []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE"},
					Type: []string{}},
				Path{Name: "serve.proxy.cors.allowed_origins", Default: []interface{}{"*"},
					Type: []string{}},
				Path{Name: "serve.proxy.cors.debug", Default: false, Type: false},
				Path{Name: "serve.proxy.cors.enabled", Default: false, Type: false},
				Path{Name: "serve.proxy.cors.exposed_headers", Default: []interface{}{"Content-Type"}, MinLength: 1,
					Type: []string{}},
				Path{Name: "serve.proxy.cors.max_age", Default: float64(0), Type: float64(0)},
				Path{Name: "serve.proxy.host", Default: "", Type: ""},
				Path{Name: "serve.proxy.port", Default: float64(4455), Type: float64(0)},
				Path{Name: "serve.proxy.timeout.idle", Default: "120s", Type: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$")},
				Path{Name: "serve.proxy.timeout.read", Default: "5s", Type: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$")},
				Path{Name: "serve.proxy.timeout.write", Default: "120s", Type: "", Pattern: regexp.MustCompile("^[0-9]+(ns|us|ms|s|m|h)$")},
				Path{Name: "serve.proxy.tls.cert.base64", Type: ""},
				Path{Name: "serve.proxy.tls.cert.path", Type: ""},
				Path{Name: "serve.proxy.tls.key.base64", Type: ""},
				Path{Name: "serve.proxy.tls.key.path", Type: ""},
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
					ReadOnly: true,
				},
				{
					Name:    "foo",
					Default: false,
					Type:    false,
				},
				{
					Name: "list",
					Type: []string{},
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
					Name: "bar",
					Type: "",
					CustomProperties: map[string]interface{}{
						"foobar": "bar",
					},
				},
				{
					Name: "foo",
					Type: false,
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

package configx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/spf13/pflag"

	"github.com/dgraph-io/ristretto"
	"github.com/knadh/koanf/parsers/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
)

func newKoanf(schemaPath string, configPath string) (*Provider, error) {
	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.StringSliceP("config", "c", []string{configPath}, "")

	k, err := New(schema, f)
	if err != nil {
		return nil, err
	}

	return k, nil
}

func setEnvs(t testing.TB, envs [][2]string) {
	for _, v := range envs {
		require.NoError(t, os.Setenv(v[0], v[1]))
		t.Cleanup(func() {
			_ = os.Unsetenv(v[0])
		})
	}
}

func BenchmarkKoanf(b *testing.B) {
	setEnvs(b, [][2]string{{"MUTATORS_HEADER_ENABLED", "true"}})
	schemaPath := path.Join("stub/benchmark/schema.config.json")
	k, err := newKoanf(schemaPath, "stub/benchmark/benchmark.yaml")
	require.NoError(b, err)

	keys := k.Koanf.Keys()
	numKeys := len(keys)

	b.Run("cache=false", func(b *testing.B) {
		var key string

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key = keys[i%numKeys]

			if k.Koanf.Get(key) == nil {
				b.Fatalf("cachedFind returned a nil value for key: %s", key)
			}
		}
	})

	b.Run("cache=true", func(b *testing.B) {
		for i, c := range []*ristretto.Config{
			{
				NumCounters: int64(numKeys),
				MaxCost:     500000,
				BufferItems: 64,
			},
			{
				NumCounters: int64(numKeys * 10),
				MaxCost:     1000000,
				BufferItems: 64,
			},
			{
				NumCounters: int64(numKeys * 10),
				MaxCost:     5000000,
				BufferItems: 64,
			},
		} {
			cache, err := ristretto.NewCache(c)
			require.NoError(b, err)

			b.Run(fmt.Sprintf("config=%d", i), func(b *testing.B) {
				var key string
				var found bool
				var val interface{}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					key = keys[i%numKeys]

					if val, found = cache.Get(key); !found {
						val = k.Koanf.Get(key)
						_ = cache.Set(key, val, 0)
					}

					if val == nil {
						b.Fatalf("cachedFind returned a nil value for key: %s", key)
					}
				}
			})
		}
	})
}

func TestKoanf(t *testing.T) {
	for _, tc := range []struct {
		stub      string
		envs      [][2]string
		isValid   bool
		expectedF func(*testing.T, *Provider)
	}{
		{stub: "kratos", isValid: true, envs: [][2]string{
			{"DSN", "sqlite:///var/lib/sqlite/db.sqlite?_fk=true"},
		}},
		{stub: "hydra", isValid: true, envs: [][2]string{
			{"DSN", "sqlite:///var/lib/sqlite/db.sqlite?_fk=true"},
			{"TRACING_PROVIDER", "jaeger"},
			{"TRACING_PROVIDERS_JAEGER_SAMPLING_SERVER_URL", "http://jaeger:5778/sampling"},
			{"TRACING_PROVIDERS_JAEGER_LOCAL_AGENT_ADDRESS", "jaeger:6832"},
			{"TRACING_PROVIDERS_JAEGER_SAMPLING_TYPE", "const"},
			{"TRACING_PROVIDERS_JAEGER_SAMPLING_VALUE", "1"},
		},
			expectedF: func(t *testing.T, p *Provider) {
				assert.Equal(t, "sqlite:///var/lib/sqlite/db.sqlite?_fk=true", p.Get("dsn"))
				assert.Equal(t, "jaeger", p.Get("tracing.provider"))
				assert.Equal(t, "jaeger:6832", p.Get("tracing.providers.jaeger.local_agent_address"))
			}},
	} {
		t.Run("service="+tc.stub, func(t *testing.T) {
			setEnvs(t, tc.envs)

			expected, err := ioutil.ReadFile(path.Join("stub", tc.stub, "expected.json"))

			schemaPath := path.Join("stub", tc.stub, "config.schema.json")
			k, err := newKoanf(schemaPath, path.Join("stub", tc.stub, tc.stub+".yaml"))
			require.NoError(t, err)

			out, err := k.Koanf.Marshal(json.Parser())
			require.NoError(t, err)

			validator, err := jsonschema.NewCompiler().Compile(schemaPath)
			require.NoError(t, err)
			err = validator.Validate(bytes.NewReader(out))
			if !tc.isValid {
				require.Error(t, err)
				return
			}

			assert.JSONEq(t, string(expected), string(out), "%s", out)

			if tc.expectedF != nil {
				tc.expectedF(t, k)
			}
		})
	}
}

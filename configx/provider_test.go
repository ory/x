package configx

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/knadh/koanf/parsers/json"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/urlx"

	"github.com/spf13/pflag"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderMethods(t *testing.T) {
	// Fake some flags
	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.String("foo-bar-baz", "", "")
	f.StringP("b", "b", "", "")
	args := []string{"/var/folders/mt/m1dwr59n73zgsq7bk0q2lrmc0000gn/T/go-build533083141/b001/exe/asdf", "aaaa", "-b", "bbbb", "dddd", "eeee", "--foo-bar-baz", "fff"}
	require.NoError(t, f.Parse(args[1:]))
	RegisterFlags(f)

	p, err := New([]byte(`{}`), f, logrusx.New("", ""))
	require.NoError(t, err)

	t.Run("check flags", func(t *testing.T) {
		assert.Equal(t, "fff", p.String("foo-bar-baz"))
		assert.Equal(t, "bbbb", p.String("b"))
	})

	t.Run("check fallbacks", func(t *testing.T) {
		t.Run("type=string", func(t *testing.T) {
			p.Set("some.string", "bar")
			assert.Equal(t, "bar", p.StringF("some.string", "baz"))
			assert.Equal(t, "baz", p.StringF("not.some.string", "baz"))
		})
		t.Run("type=float", func(t *testing.T) {
			p.Set("some.float", 123.123)
			assert.Equal(t, 123.123, p.Float64F("some.float", 321.321))
			assert.Equal(t, 321.321, p.Float64F("not.some.float", 321.321))
		})
		t.Run("type=int", func(t *testing.T) {
			p.Set("some.int", 123)
			assert.Equal(t, 123, p.IntF("some.int", 123))
			assert.Equal(t, 321, p.IntF("not.some.int", 321))
		})

		github := urlx.ParseOrPanic("https://github.com/ory")
		ory := urlx.ParseOrPanic("https://www.ory.sh/")

		t.Run("type=url", func(t *testing.T) {
			p.Set("some.url", "https://github.com/ory")
			assert.Equal(t, github, p.URIF("some.url", ory))
			assert.Equal(t, ory, p.URIF("not.some.url", ory))
		})

		t.Run("type=request_uri", func(t *testing.T) {
			p.Set("some.request_uri", "https://github.com/ory")
			assert.Equal(t, github, p.RequestURIF("some.request_uri", ory))
			assert.Equal(t, ory, p.RequestURIF("not.some.request_uri", ory))

			p.Set("invalid.request_uri", "foo")
			assert.Equal(t, ory, p.RequestURIF("invalid.request_uri", ory))
		})
	})
}

func TestAdvancedConfigs(t *testing.T) {
	for _, tc := range []struct {
		stub      string
		configs   []string
		envs      [][2]string
		isValid   bool
		expectedF func(*testing.T, *Provider)
	}{
		{
			stub:    "kratos",
			configs: []string{"stub/kratos/kratos.yaml"},
			isValid: true, envs: [][2]string{
				{"DSN", "sqlite:///var/lib/sqlite/db.sqlite?_fk=true"},
			}},
		{
			stub:    "kratos",
			configs: []string{"stub/kratos/multi/a.yaml", "stub/kratos/multi/b.yaml"},
			isValid: true, envs: [][2]string{
				{"DSN", "sqlite:///var/lib/sqlite/db.sqlite?_fk=true"},
			}},
		{
			stub:    "hydra",
			configs: []string{"stub/hydra/hydra.yaml"},
			isValid: true,
			envs: [][2]string{
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
		{
			stub:    "hydra",
			configs: []string{"stub/hydra/hydra.yaml"},
			isValid: false, envs: [][2]string{
				{"DSN", "sqlite:///var/lib/sqlite/db.sqlite?_fk=true"},
				{"TRACING_PROVIDER", "not-jaeger"},
			}},
	} {
		t.Run("service="+tc.stub, func(t *testing.T) {
			setEnvs(t, tc.envs)

			expected, err := ioutil.ReadFile(path.Join("stub", tc.stub, "expected.json"))
			require.NoError(t, err)

			schemaPath := path.Join("stub", tc.stub, "config.schema.json")
			k, err := newKoanf(schemaPath, tc.configs, logrusx.New("", ""))
			if !tc.isValid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			out, err := k.Koanf.Marshal(json.Parser())
			require.NoError(t, err)
			assert.JSONEq(t, string(expected), string(out), "%s", out)

			if tc.expectedF != nil {
				tc.expectedF(t, k)
			}
		})
	}
}

package viperx

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/viper"

	"github.com/ory/x/logrusx"
)

func TestViperInit(t *testing.T) {
	l := logrusx.New("", "")
	l.Entry.Logger.ExitFunc = func(code int) {
		panic(code)
	}

	t.Run("suite=home-path", func(t *testing.T) {
		for k, tc := range []struct {
			h string
			e int
		}{
			{h: "./stub/json", e: 1},
			{h: "./stub/toml", e: 2},
			{h: "./stub/yaml", e: 3},
			{h: "./stub/yml", e: 4},
			{h: "./stub/does-not-exist/", e: 0},
			{h: "./stub/", e: 0},
		} {
			t.Run(fmt.Sprintf("case=%d/path=%s", k, tc.h), func(t *testing.T) {
				viper.Reset()

				path, err := filepath.Abs(tc.h)
				require.NoError(t, err)

				InitializeConfig("project-stub-name", path, l)
				assert.Equal(t, tc.e, viper.GetInt("serve.admin.port"))
			})
		}
	})

	t.Run("suite=with-config-path", func(t *testing.T) {
		for k, tc := range []struct {
			f      string
			fatals bool
		}{
			{f: "./stub/json/.project-stub-name.json"},
			{f: "./stub/toml/.project-stub-name.toml"},
			{f: "./stub/yaml/.project-stub-name.yaml"},
			{f: "./stub/yml/.project-stub-name.yml"},
			{f: "./stub/does-not-exist/foo.yml", fatals: true},
		} {
			t.Run(fmt.Sprintf("case=%d/path=%s", k, tc.f), func(t *testing.T) {
				viper.Reset()

				cfgFile = tc.f
				if tc.fatals {
					assert.Panics(t, func() {
						InitializeConfig("project-stub-name", "", l)
					})
				} else {
					InitializeConfig("project-stub-name", "", l)
					assert.Equal(t, k+1, viper.GetInt("serve.admin.port"))
				}
			})
		}
	})

	t.Run("suite=os-env", func(t *testing.T) {
		for k, tc := range []struct {
			n string
			v int
			e int
		}{
			{
				n: "serve.admin.port",
				v: 5,
			},
			{
				n: "SERVE.ADMIN.PORT",
				v: 5,
			},
			{
				n: "SERVE_ADMIN_PORT",
				v: 5,
				e: 5,
			},
		} {
			t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
				viper.Reset()
				cfgFile = ""
				InitializeConfig(uuid.New().String(), "", l)
				require.NoError(t, os.Setenv(tc.n, fmt.Sprintf("%d", tc.v)))
				assert.Equal(t, tc.e, viper.GetInt("serve.admin.port"))
			})
		}
	})
}

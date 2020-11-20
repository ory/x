package configx

import (
	"path/filepath"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/pkg/errors"
	"github.com/rs/cors"
)

type Provider struct {
	k *koanf.Koanf
	c *ristretto.Cache
}

func New(schema []byte, configFiles ...string) (*Provider, error) {
	p := &Provider{k: koanf.New(".")}

	dp, err := NewKoanfSchemaDefaults(schema)
	if err != nil {
		return nil, err
	}

	ep, err := NewKoanfEnv("", schema)
	if err != nil {
		return nil, err
	}

	// Load defaults
	if err := p.k.Load(dp, nil); err != nil {
		return nil, err
	}

	for _, configFile := range configFiles {
		if err := p.addConfigFile(configFile); err != nil {
			return nil, err
		}
	}

	if err := p.k.Load(ep, nil); err != nil {
		return nil, err
	}

	p.newCache()
	return p, nil
}

func (p *Provider) newCache() {
	old := p.c

	// This can not error as all config values are > 0
	c, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: int64(len(p.k.Keys()) * 10),
		MaxCost:     5000000,
		BufferItems: 64,
	})

	p.c = c

	if old != nil {
		old.Close()
	}
}

func (p *Provider) addConfigFile(path string) error {
	var parser koanf.Parser

	switch e := filepath.Ext(path); e {
	case ".toml":
		parser = toml.Parser()
	case ".json":
		parser = json.Parser()
	case ".yaml", ".yml":
		parser = yaml.Parser()
	default:
		return errors.Errorf("unknown config file extension: %s", e)
	}

	return p.k.Load(file.Provider(path), parser)
}

func (p *Provider) Koanf() *koanf.Koanf {
	return p.k
}

func (p *Provider) Set(key string, value interface{}) {
	// This can not err because confmap does not err
	_ = p.k.Load(confmap.Provider(map[string]interface{}{
		key: value}, "."), nil)
	p.newCache()
}

func (p *Provider) Bool(key string) bool {
	return p.BoolF(key, false)
}

func (p *Provider) BoolF(key string, fallback bool) bool {
	if !p.k.Exists(key) {
		return fallback
	}

	return p.k.Bool(key)
}

func (p *Provider) String(key string) string {
	return p.StringF(key, "")
}

func (p *Provider) StringF(key string, fallback string) string {
	if !p.k.Exists(key) {
		return fallback
	}

	return p.k.String(key)
}

func (p *Provider) Strings(key string) []string {
	return p.StringsF(key, []string{})
}

func (p *Provider) StringsF(key string, fallback []string) (val []string) {
	if !p.k.Exists(key) {
		return fallback
	}

	return p.k.Strings(key)
}

func (p *Provider) Int(key string) int {
	return p.IntF(key, 0)
}

func (p *Provider) IntF(key string, fallback int) (val int) {
	if !p.k.Exists(key) {
		return fallback
	}

	return p.k.Int(key)
}

func (p *Provider) Get(key string) bool {
	return p.BoolF(key, false)
}

func (p *Provider) GetF(key string, fallback interface{}) (val interface{}) {
	val = p.k.Get(key)
	if val == nil {
		return fallback
	}

	return val
}

func (p *Provider) CORS(prefix string, defaults cors.Options) (cors.Options, bool) {
	if len(prefix) > 0 {
		prefix = strings.TrimRight(prefix, ".") + "."
	}

	return cors.Options{
		AllowedOrigins:     p.StringsF(prefix+"cors.allowed_origins", defaults.AllowedOrigins),
		AllowedMethods:     p.StringsF(prefix+"cors.allowed_methods", defaults.AllowedMethods),
		AllowedHeaders:     p.StringsF(prefix+"cors.allowed_headers", defaults.AllowedHeaders),
		ExposedHeaders:     p.StringsF(prefix+"cors.exposed_headers", defaults.ExposedHeaders),
		AllowCredentials:   p.BoolF(prefix+"cors.allow_credentials", defaults.AllowCredentials),
		OptionsPassthrough: p.BoolF(prefix+"cors.options_passthrough", defaults.OptionsPassthrough),
		MaxAge:             p.IntF(prefix+"cors.max_age", defaults.MaxAge),
		Debug:              p.BoolF(prefix+"cors.debug", defaults.Debug),
	}, p.Bool(prefix + "cors.enabled")
}

//func (p *Provider) Get(key string) (val interface{}) {
//	return p.GetF(key, nil)
//}
//
//func (p *Provider) GetF(key string, fallback interface{}) (val interface{}) {
//	var found bool
//
//	if p.c == nil {
//		val = p.k.Get(key)
//		if val == nil {
//			return fallback
//		}
//
//		return val
//	}
//
//
//	val, found = p.c.Get(key)
//	if found {
//		return val
//	}
//
//	val = p.k.Get(key)
//	_ = p.c.Set(key, val, 0)
//
//	if val == nil {
//		return fallback
//	}
//
//	return val
//}

package configx

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/pflag"

	"github.com/ory/x/stringsx"
	"github.com/ory/x/tracing"

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
	*koanf.Koanf
}

// RegisterFlags registers the config file flag.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.StringSliceP("config", "c", []string{}, "Path to one or more .json, .yaml, .yml, .toml config files. Values are loaded in the order provided, meaning that the last config file overwrites values from the previous config file.")
}

// New creates a new provider instance or errors.
// Configuration values are loaded in the following order:
//
// 1. Defaults from the JSON Schema
// 2. Config files (yaml, yml, toml, json)
// 3. Command line flags
// 4. Environment variables
func New(schema []byte, flags *pflag.FlagSet) (*Provider, error) {
	p := &Provider{Koanf: koanf.New(".")}

	dp, err := NewKoanfSchemaDefaults(schema)
	if err != nil {
		return nil, err
	}

	ep, err := NewKoanfEnv("", schema)
	if err != nil {
		return nil, err
	}

	// Load defaults
	if err := p.Koanf.Load(dp, nil); err != nil {
		return nil, err
	}

	paths, err := flags.GetStringSlice("config")
	for _, configFile := range paths {
		if err := p.addConfigFile(configFile); err != nil {
			return nil, err
		}
	}

	if err := p.Koanf.Load(posflag.Provider(flags, ".", p.Koanf), nil); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	if err := p.Koanf.Load(ep, nil); err != nil {
		return nil, err
	}

	return p, nil
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

	return p.Koanf.Load(file.Provider(path), parser)
}

func (p *Provider) Set(key string, value interface{}) {
	// This can not err because confmap does not err
	_ = p.Koanf.Load(confmap.Provider(map[string]interface{}{
		key: value}, "."), nil)
}

func (p *Provider) BoolF(key string, fallback bool) bool {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Bool(key)
}

func (p *Provider) StringF(key string, fallback string) string {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.String(key)
}

func (p *Provider) StringsF(key string, fallback []string) (val []string) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Strings(key)
}

func (p *Provider) IntF(key string, fallback int) (val int) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Int(key)
}

func (p *Provider) Float64F(key string, fallback float64) (val float64) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Float64(key)
}

func (p *Provider) DurationF(key string, fallback time.Duration) (val time.Duration) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Duration(key)
}

func (p *Provider) GetF(key string, fallback interface{}) (val interface{}) {
	val = p.Get(key)
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

func (p *Provider) TracingConfig(serviceName string) *tracing.Config {
	return &tracing.Config{
		ServiceName: p.StringF("tracing.service_name", serviceName),
		Provider:    p.String("tracing.provider"),
		Jaeger: &tracing.JaegerConfig{
			LocalAgentHostPort: p.String("tracing.providers.jaeger.local_agent_address"),
			SamplerType:        p.StringF("tracing.providers.jaeger.sampling.type", "const"),
			SamplerValue:       p.Float64F("tracing.providers.jaeger.sampling.value", float64(1)),
			SamplerServerURL:   p.String("tracing.providers.jaeger.sampling.server_url"),
			Propagation: stringsx.Coalesce(
				os.Getenv("JAEGER_PROPAGATION"),
				p.String("tracing.providers.jaeger.propagation"),
			),
		},
		Zipkin: &tracing.ZipkinConfig{
			ServerURL: p.String("tracing.providers.zipkin.server_url"),
		},
	}
}

func (p *Provider) RequestURIF(path string, fallback *url.URL) *url.URL {
	if p.Get(path) == nil {
		return fallback
	}

	parsed, err := url.ParseRequestURI(p.String(path))
	if err != nil {
		return fallback
	}

	return parsed
}

//func (p *Provider) Get(key string) (val interface{}) {
//	return p.GetF(key, nil)
//}
//
//func (p *Provider) GetF(key string, fallback interface{}) (val interface{}) {
//	var found bool
//
//	if p.c == nil {
//		val = p.Koanf.Get(key)
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
//	val = p.Koanf.Get(key)
//	_ = p.c.Set(key, val, 0)
//
//	if val == nil {
//		return fallback
//	}
//
//	return val
//}

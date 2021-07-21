package tracing

import (
	"bytes"
	_ "embed"
	"io"
)

// JaegerConfig encapsulates jaeger's configuration.
type JaegerConfig struct {
	LocalAgentAddress string          `json:"local_agent_address"`
	Sampling          *JaegerSampling `json:"sampling"`
	Propagation       string          `json:"propagation"`
	MaxTagValueLength int             `json:"max_tag_value_length"`
}

type JaegerSampling struct {
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	ServerURL string  `json:"server_url"`
}

// ZipkinConfig encapsulates zipkin's configuration.
type ZipkinConfig struct {
	ServerURL string `json:"server_url"`
}

type Config struct {
	ServiceName string           `json:"service_name"`
	Provider    string           `json:"provider"`
	Providers   *ProvidersConfig `json:"providers"`
}

type ProvidersConfig struct {
	Jaeger *JaegerConfig `json:"jaeger"`
	Zipkin *ZipkinConfig `json:"zipkin"`
}

//go:embed config.schema.json
var ConfigSchema string

const ConfigSchemaID = "ory://tracing-config"

// AddConfigSchema adds the tracing schema to the compiler.
// The interface is specified instead of `jsonschema.Compiler` to allow the use of any jsonschema library fork or version.
func AddConfigSchema(c interface {
	AddResource(url string, r io.Reader) error
}) error {
	return c.AddResource(ConfigSchemaID, bytes.NewBufferString(ConfigSchema))
}

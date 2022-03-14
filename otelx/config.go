package otelx

import (
	"bytes"
<<<<<<< HEAD
	_ "embed"
=======
>>>>>>> da7ab92 (feat: add otelx package)
	"io"
)

type JaegerConfig struct {
<<<<<<< HEAD
	LocalAgentHost string  `json:"local_agent_host"`
	LocalAgentPort int     `json:"local_agent_port"`
	SamplingRatio  float64 `json:"sampling_ratio"`
=======
	LocalAgentAddress string  `json:"local_agent_address"`
	SamplingRatio     float64 `json:"sampling_ratio"`
>>>>>>> da7ab92 (feat: add otelx package)
}

type ProvidersConfig struct {
	Jaeger *JaegerConfig `json:"jaeger"`
}

type Config struct {
	ServiceName string           `json:"service_name"`
	Provider    string           `json:"provider"`
	Providers   *ProvidersConfig `json:"providers"`
}

//go:embed config.schema.json
var ConfigSchema string

const ConfigSchemaID = "ory://otelx-config"

// AddConfigSchema adds the tracing schema to the compiler.
// The interface is specified instead of `jsonschema.Compiler` to allow the use of any jsonschema library fork or version.
func AddConfigSchema(c interface {
	AddResource(url string, r io.Reader) error
}) error {
	return c.AddResource(ConfigSchemaID, bytes.NewBufferString(ConfigSchema))
}

package tracing

// JaegerConfig encapsulates jaeger's configuration.
type JaegerConfig struct {
	LocalAgentHostPort string
	SamplerType        string
	SamplerValue       float64
	SamplerServerURL   string
	Propagation        string
	MaxTagValueLength  int
}

// ZipkinConfig encapsulates zipkin's configuration.
type ZipkinConfig struct {
	ServerURL string
}

type Config struct {
	ServiceName string
	Provider    string
	Jaeger      *JaegerConfig
	Zipkin      *ZipkinConfig
}

package tracing

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"

	"github.com/ory/x/logrusx"

	zipkinOT "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	zipkinHttp "github.com/openzipkin/zipkin-go/reporter/http"

	jaegerConf "github.com/uber/jaeger-client-go/config"
	jaegerZipkin "github.com/uber/jaeger-client-go/zipkin"

	datadogOpentracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	datadogTracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmot"
)

// Tracer encapsulates tracing abilities.
type Tracer struct {
	ServiceName  string
	Provider     string
	Logger       *logrusx.Logger
	JaegerConfig *JaegerConfig
	ZipkinConfig *ZipkinConfig

	tracer opentracing.Tracer
	closer io.Closer
}

// JaegerConfig encapsulates jaeger's configuration.
type JaegerConfig struct {
	LocalAgentHostPort string
	SamplerType        string
	SamplerValue       float64
	SamplerServerURL   string
	Propagation        string
}

// ZipkinConfig encapsulates zipkin's configuration.
type ZipkinConfig struct {
	ServerURL string
}

type datadogCloser struct {
}

func (t datadogCloser) Close() error {
	datadogTracer.Stop()
	return nil
}

// Setup sets up the tracer. Currently supports jaeger.
func (t *Tracer) Setup() error {
	switch strings.ToLower(t.Provider) {
	case "jaeger":
		jc, err := jaegerConf.FromEnv()

		if err != nil {
			return err
		}

		if t.JaegerConfig.SamplerServerURL != "" {
			jc.Sampler.SamplingServerURL = t.JaegerConfig.SamplerServerURL
		}

		if t.JaegerConfig.SamplerType != "" {
			jc.Sampler.Type = t.JaegerConfig.SamplerType
		}

		if t.JaegerConfig.SamplerValue != 0 {
			jc.Sampler.Param = t.JaegerConfig.SamplerValue
		}

		if t.JaegerConfig.LocalAgentHostPort != "" {
			jc.Reporter.LocalAgentHostPort = t.JaegerConfig.LocalAgentHostPort
		}

		var configs []jaegerConf.Option

		// This works in other jaeger clients, but is not part of jaeger-client-go
		if t.JaegerConfig.Propagation == "b3" {
			zipkinPropagator := jaegerZipkin.NewZipkinB3HTTPHeaderPropagator()
			configs = append(
				configs,
				jaegerConf.Injector(opentracing.HTTPHeaders, zipkinPropagator),
				jaegerConf.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
			)
		}

		closer, err := jc.InitGlobalTracer(
			t.ServiceName,
			configs...,
		)

		if err != nil {
			return err
		}

		t.closer = closer
		t.tracer = opentracing.GlobalTracer()
		t.Logger.Infof("Jaeger tracer configured!")
	case "zipkin":
		if t.ZipkinConfig.ServerURL == "" {
			return errors.Errorf("Zipkin's server url is required")
		}

		reporter := zipkinHttp.NewReporter(t.ZipkinConfig.ServerURL)

		endpoint, err := zipkin.NewEndpoint(t.ServiceName, "")

		if err != nil {
			return err
		}

		nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))

		if err != nil {
			return err
		}

		opentracing.SetGlobalTracer(zipkinOT.Wrap(nativeTracer))

		t.closer = reporter
		t.tracer = opentracing.GlobalTracer()
		t.Logger.Infof("Zipkin tracer configured!")
	case "datadog":
		var serviceName = os.Getenv("DD_SERVICE")
		if serviceName == "" {
			serviceName = t.ServiceName
		}

		opentracing.SetGlobalTracer(datadogOpentracer.New(datadogTracer.WithService(serviceName)))

		t.closer = datadogCloser{}
		t.tracer = opentracing.GlobalTracer()
		t.Logger.Infof("DataDog tracer configured!")
	case "elastic-apm":
		var serviceName = os.Getenv("ELASTIC_APM_SERVICE_NAME")
		if serviceName == "" {
			serviceName = t.ServiceName
		}

		tr, err := apm.NewTracer(serviceName, "")
		if err != nil {
			log.Fatal(err)
		}
		opentracing.SetGlobalTracer(apmot.New(apmot.WithTracer(tr)))

		//t.closer = tr.Close
		t.tracer = opentracing.GlobalTracer()
		t.Logger.Infof("Elastic APM tracer configured!")

	case "":
		t.Logger.Infof("No tracer configured - skipping tracing setup")
	default:
		return errors.Errorf("unknown tracer: %s", t.Provider)
	}
	return nil
}

// IsLoaded returns true if the tracer has been loaded.
func (t *Tracer) IsLoaded() bool {
	if t == nil || t.tracer == nil {
		return false
	}
	return true
}

// Close closes the tracer.
func (t *Tracer) Close() {
	if t.closer != nil {
		err := t.closer.Close()
		if err != nil {
			t.Logger.Warn(err)
		}
	}
}

// HelpMessage returns a help message for CLIs using tracing.
func HelpMessage(defaultName string) string {
	return fmt.Sprintf(`- TRACING_PROVIDER: Set this to the tracing backend you wish to use. Currently supported tracing backends:
		- "": No tracing enabled (default)
		- "jaeger": Enables Jaeger tracing
		- "zipkin": Enables Zipkin tracing
		- "datadog": Enables Datadog tracing
		- "elastic-apm": Enables elastic apm tracing

	Example: TRACING_PROVIDER=jaeger

- TRACING_PROVIDERS_JAEGER_SAMPLING_SERVER_URL: The address of Jaeger-agent's HTTP sampling server.

	Example: TRACING_PROVIDERS_JAEGER_SAMPLING_SERVER_URL=http://localhost:5778/sampling

- TRACING_PROVIDERS_JAEGER_SAMPLING_TYPE: The type of the sampler you want to use. Supported values:
		- const (default)
		- probabilistic
		- ratelimiting

	Example: TRACING_PROVIDER_JAEGER_SAMPLING_TYPE=const

- TRACING_PROVIDERS_JAEGER_SAMPLING_VALUE: The value passed to the sampler type that has been configured.

	Supported values (dependant on sampling strategy used):
		- const: 0 or 1 (all or nothing)
		- rateLimiting: a constant rate (e.g. setting this to 3 will sample requests with the rate of 3 traces per second)
		- probabilistic: a value between 0..1

	Example: TRACING_PROVIDERS_JAEGER_SAMPLING_VALUE=1

- TRACING_PROVIDERS_JAEGER_LOCAL_AGENT_ADDRESS: The address of the jaeger-agent where spans should be sent to

	Example: TRACING_PROVIDERS_JAEGER_LOCAL_AGENT_ADDRESS=127.0.0.1:6831

- TRACING_PROVIDERS_JAEGER_PROPAGATION: The tracing header propagation format. Defaults to jaeger.

	Example: TRACING_PROVIDERS_JAEGER_PROPAGATION=b3

- TRACING_PROVIDERS_ZIPKIN_SERVER_URL: The address of Zipkin server

	Example: TRACING_PROVIDERS_ZIPKIN_SERVER_URL=http://localhost:9411/api/v2/spans

- TRACING_SERVICE_NAME: Specifies the service name to use on the tracer.
	Default: ORY Hydra

	Example: TRACING_SERVICE_NAME="%s"
`, defaultName)
}

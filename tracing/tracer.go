package tracing

import (
	"context"
	"io"
	"os"
	"strings"

	instana "github.com/instana/go-sensor"
	"github.com/uber/jaeger-client-go"

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

	"go.opentelemetry.io/otel"
	otelOpentracing "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	otelSdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmot"
)

// Tracer encapsulates tracing abilities.
type Tracer struct {
	Config *Config

	l      *logrusx.Logger
	tracer opentracing.Tracer
	closer io.Closer
}

func New(l *logrusx.Logger, c *Config) (*Tracer, error) {
	t := &Tracer{Config: c, l: l}

	if err := t.setup(); err != nil {
		return nil, err
	}

	return t, nil
}

// setup sets up the tracer. Currently supports jaeger.
func (t *Tracer) setup() error {
	switch strings.ToLower(t.Config.Provider) {
	case "jaeger":
		jc, err := jaegerConf.FromEnv()

		if err != nil {
			return err
		}

		if t.Config.Providers.Jaeger.Sampling.ServerURL != "" {
			jc.Sampler.SamplingServerURL = t.Config.Providers.Jaeger.Sampling.ServerURL
		}

		if t.Config.Providers.Jaeger.Sampling.Type != "" {
			jc.Sampler.Type = t.Config.Providers.Jaeger.Sampling.Type
		}

		if t.Config.Providers.Jaeger.Sampling.Value != 0 {
			jc.Sampler.Param = t.Config.Providers.Jaeger.Sampling.Value
		}

		if t.Config.Providers.Jaeger.LocalAgentAddress != "" {
			jc.Reporter.LocalAgentHostPort = t.Config.Providers.Jaeger.LocalAgentAddress
		}

		var configs []jaegerConf.Option

		if t.Config.Providers.Jaeger.MaxTagValueLength != jaeger.DefaultMaxTagValueLength {
			configs = append(configs, jaegerConf.MaxTagValueLength(t.Config.Providers.Jaeger.MaxTagValueLength))
		}

		// This works in other jaeger clients, but is not part of jaeger-client-go
		if t.Config.Providers.Jaeger.Propagation == "b3" {
			zipkinPropagator := jaegerZipkin.NewZipkinB3HTTPHeaderPropagator()
			configs = append(
				configs,
				jaegerConf.Injector(opentracing.HTTPHeaders, zipkinPropagator),
				jaegerConf.Extractor(opentracing.HTTPHeaders, zipkinPropagator),
			)
		}

		closer, err := jc.InitGlobalTracer(
			t.Config.ServiceName,
			configs...,
		)

		if err != nil {
			return err
		}

		t.closer = closer
		t.tracer = opentracing.GlobalTracer()
		t.l.Infof("Jaeger tracer configured!")
	case "zipkin":
		if t.Config.Providers.Zipkin.ServerURL == "" {
			return errors.Errorf("Zipkin's server url is required")
		}

		reporter := zipkinHttp.NewReporter(t.Config.Providers.Zipkin.ServerURL)

		endpoint, err := zipkin.NewEndpoint(t.Config.ServiceName, "")

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
		t.l.Infof("Zipkin tracer configured!")
	case "datadog":
		var serviceName = os.Getenv("DD_SERVICE")
		if serviceName == "" {
			serviceName = t.Config.ServiceName
		}

		opentracing.SetGlobalTracer(datadogOpentracer.New(datadogTracer.WithService(serviceName)))

		t.closer = datadogCloser{}
		t.tracer = opentracing.GlobalTracer()
		t.l.Infof("DataDog tracer configured!")
	case "elastic-apm":
		var serviceName = os.Getenv("ELASTIC_APM_SERVICE_NAME")
		if serviceName == "" {
			serviceName = t.Config.ServiceName
		}

		tr, err := apm.NewTracer(serviceName, "")
		if err != nil {
			return err
		}
		opentracing.SetGlobalTracer(apmot.New(apmot.WithTracer(tr)))

		//t.closer = tr.Close
		t.tracer = opentracing.GlobalTracer()
		t.l.Infof("Elastic APM tracer configured!")

	case "instana":
		opts := instana.DefaultOptions()
		var serviceName = os.Getenv("INSTANA_SERVICE_NAME")
		if serviceName == "" {
			serviceName = t.Config.ServiceName
		}
		opts.Service = serviceName
		// all other settings can be configured using environment variables

		t.tracer = instana.NewTracerWithOptions(opts)
		opentracing.SetGlobalTracer(t.tracer)

		t.l.Infof("Instana tracer configured!")
	case "otel":
		ctx := context.Background()
		var serviceName = os.Getenv("OTEL_SERVICE_NAME")
		if serviceName == "" {
			serviceName = t.Config.ServiceName
		}

		res, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.ServiceNameKey.String(serviceName),
			),
		)
		if err != nil {
			return errors.Wrap(err, "new otel resource")
		}

		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return errors.Wrap(err, "new otel exporter")
		}

		tp := otelSdkTrace.NewTracerProvider(
			otelSdkTrace.WithResource(res),
			otelSdkTrace.WithSpanProcessor(
				otelSdkTrace.NewSimpleSpanProcessor(exporter),
			),
		)

		otel.SetTracerProvider(tp)

		bridge := otelOpentracing.NewBridgeTracer()
		bridge.SetOpenTelemetryTracer(otel.Tracer(""))

		t.tracer = bridge
		opentracing.SetGlobalTracer(t.tracer)

		t.l.Infof("OTEL tracer configured!")
	case "":
		t.l.Infof("No tracer configured - skipping tracing setup")
	default:
		return errors.Errorf("unknown tracer: %s", t.Config.Provider)
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

// Tracer returns the wrapped tracer
func (t *Tracer) Tracer() opentracing.Tracer {
	return t.tracer
}

// Close closes the tracer.
func (t *Tracer) Close() {
	if t.closer != nil {
		err := t.closer.Close()
		if err != nil {
			t.l.WithError(err).Error("Unable to close tracer.")
		}
	}
}

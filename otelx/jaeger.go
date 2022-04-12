package otelx

import (
	"os"
	"strconv"
	"strings"

	"github.com/ory/x/stringsx"
	"go.opentelemetry.io/contrib/propagators/b3"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/contrib/samplers/jaegerremote"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func configureHostPort(c Config) (host, port string) {
	address := stringsx.Coalesce(
		c.Providers.Jaeger.LocalAgentAddress,
		os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_ADDRESS"),
	)
	splitAddr := strings.Split(address, ":")

	if len(splitAddr) == 2 {
		host = splitAddr[0]
		port = splitAddr[1]
	} else {
		host = stringsx.Coalesce(
			c.Providers.Jaeger.LocalAgentHost,
			os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_HOST"),
		)

		if c.Providers.Jaeger.LocalAgentPort != 0 {
			port = strconv.Itoa(c.Providers.Jaeger.LocalAgentPort)
		} else {
			port = os.Getenv("OTEL_EXPORTER_JAEGER_AGENT_PORT")
		}
	}
	return
}

// Endpoint configuration is implicitly read from the below environment
// variables, by default:
//
//    OTEL_EXPORTER_JAEGER_AGENT_ADDRESS (takes precedence)
//    OTEL_EXPORTER_JAEGER_AGENT_HOST
//    OTEL_EXPORTER_JAEGER_AGENT_PORT
//    OTEL_EXPORTER_JAEGER_SAMPLING_SERVER_URL
//
// Optionally, Config.Providers.Jaeger.LocalAgentAddress can be set.
// NOTE: If Config.Providers.Jaeger.Sampling.ServerURL is not specfied,
// AlwaysSample is used.
func SetupJaeger(t *Tracer, tracerName string) (trace.Tracer, error) {
	host, port := configureHostPort(*t.Config)
	exp, err := jaeger.New(
		jaeger.WithAgentEndpoint(
			jaeger.WithAgentHost(host), jaeger.WithAgentPort(port),
		),
	)
	if err != nil {
		return nil, err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(t.Config.ServiceName),
		)),
	}

	samplingServerURL := stringsx.Coalesce(
		t.Config.Providers.Jaeger.Sampling.ServerURL,
		os.Getenv("OTEL_EXPORTER_JAEGER_SAMPLING_SERVER_URL"),
	)

	if samplingServerURL != "" {
		jaegerRemoteSampler := jaegerremote.New(
			"jaegerremote",
			jaegerremote.WithSamplingServerURL(samplingServerURL),
		)
		tpOpts = append(tpOpts, sdktrace.WithSampler(jaegerRemoteSampler))
	} else {
		tpOpts = append(tpOpts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	// At the moment, software across our cloud stack only support Zipkin (B3)
	// and Jaeger propagation formats. Proposals for standardized formats for
	// context propagation are in the works (ref: https://www.w3.org/TR/trace-context/
	// and https://www.w3.org/TR/baggage/).
	//
	// Simply add propagation.TraceContext{} and propagation.Baggage{}
	// here to enable those as well.
	prop := propagation.NewCompositeTextMapPropagator(
		jaegerPropagator.Jaeger{},
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)),
	)
	otel.SetTextMapPropagator(prop)
	return tp.Tracer(tracerName), nil
}

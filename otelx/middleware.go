package otelx

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const tracingComponent = "github.com/ory/x/otelx"

func isHealthFilter(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/health/") {
		return false
	}
	return true
}

func filterOpts() []otelhttp.Option {
	filters := []otelhttp.Filter{
		isHealthFilter,
	}
	opts := []otelhttp.Option{}
	for _, f := range filters {
		opts = append(opts, otelhttp.WithFilter(f))
	}
	return opts
}

func NewHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation, filterOpts()...)
}

// Middleware to satisfy httprouter.Handle. The URL path is used as
// the span name.
func WrapHTTPRouter(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		operation := r.URL.Path
		var tracer trace.Tracer
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			tracer = span.TracerProvider().Tracer(tracingComponent)
		} else {
			tracer = otel.GetTracerProvider().Tracer(tracingComponent)
		}

		opts := append([]trace.SpanStartOption{
			trace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
			trace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
			trace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(operation, "", r)...),
		})

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		_, span := tracer.Start(ctx, operation, opts...)
		span.SetName(operation)
		defer span.End()

		r = r.WithContext(ctx)
		next(w, r, ps)
	}
}

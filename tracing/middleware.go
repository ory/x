// Package tracing provides helpers for dealing with Open Tracing and Distributed Tracing.
package tracing

import (
	"net/http"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/urfave/negroni"
)

func (t *Tracer) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var span opentracing.Span
	opName := r.URL.Path

	// Omit health endpoints
	if strings.HasPrefix(opName, "/health/") {
		next(rw, r)
		return
	}

	// It's very possible that Hydra is fronted by a proxy which could have initiated a trace.
	// If so, we should attempt to join it.
	carrier := opentracing.HTTPHeadersCarrier(r.Header)
	remoteContext, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
	span = opentracing.StartSpan(opName, ext.RPCServerOption(remoteContext))

	defer span.Finish()

	ext.HTTPMethod.Set(span, r.Method)
	ext.HTTPUrl.Set(span, r.URL.String())
	ext.Component.Set(span, t.Config.ServiceName)
	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), span))

	next(rw, r)

	if negroniWriter, ok := rw.(negroni.ResponseWriter); ok {
		statusCode := uint16(negroniWriter.Status())
		if statusCode >= 400 {
			ext.Error.Set(span, true)
		}
		ext.HTTPStatusCode.Set(span, statusCode)
	}
}

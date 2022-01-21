package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"net/http"
)

// The RoundTripperFunc type is an adapter to allow the use of ordinary
// functions as RoundTrippers. If f is a function with the appropriate
// signature, RountTripperFunc(f) is a RoundTripper that calls f.
type RoundTripperFunc func(req *http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface.
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt(r)
}

func RoundTripper(tracer opentracing.Tracer, delegate http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		ctx := req.Context()

		saveURL := *req.URL
		saveURL.RawQuery = ""
		saveURL.User = nil

		span, _ := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "webhook", ext.SpanKindRPCClient,
			opentracing.Tags{
				"http.url":    saveURL.String(),
				"http.method": req.Method,
			})
		defer span.Finish()
		carrier := opentracing.HTTPHeadersCarrier(req.Header)
		span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, carrier)

		if delegate == nil {
			delegate = http.DefaultTransport
		}

		resp, err := delegate.RoundTrip(req)

		if err != nil {
			span.SetTag("http.error", err.Error())
			span.LogFields(otlog.Error(err))
			return resp, err
		}
		ext.HTTPStatusCode.Set(span, uint16(resp.StatusCode))
		return resp, err
	})
}

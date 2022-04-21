package tracing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/negroni"

	"github.com/ory/x/tracing"
)

var mockedTracer *mocktracer.MockTracer
var tracer = &tracing.Tracer{Config: &tracing.Config{
	ServiceName: "Ory Hydra Test",
	Provider:    "Mock Provider",
}}

func init() {
	mockedTracer = mocktracer.New()
	opentracing.SetGlobalTracer(mockedTracer)
}

func TestTracingServeHttp(t *testing.T) {
	expectedTagsSuccess := map[string]interface{}{
		string(ext.Component):      "Ory Hydra Test",
		string(ext.SpanKind):       ext.SpanKindRPCServerEnum,
		string(ext.HTTPMethod):     "GET",
		string(ext.HTTPUrl):        "https://apis.somecompany.com/endpoint",
		string(ext.HTTPStatusCode): uint16(200),
	}

	expectedTagsError := map[string]interface{}{
		string(ext.Component):      "Ory Hydra Test",
		string(ext.SpanKind):       ext.SpanKindRPCServerEnum,
		string(ext.HTTPMethod):     "GET",
		string(ext.HTTPUrl):        "https://apis.somecompany.com/endpoint",
		string(ext.HTTPStatusCode): uint16(400),
		string(ext.Error):          true,
	}

	testCases := []struct {
		httpStatus      int
		testDescription string
		expectedTags    map[string]interface{}
	}{
		{
			testDescription: "success http response",
			httpStatus:      http.StatusOK,
			expectedTags:    expectedTagsSuccess,
		},
		{
			testDescription: "error http response",
			httpStatus:      http.StatusBadRequest,
			expectedTags:    expectedTagsError,
		},
	}

	for _, test := range testCases {
		t.Run(test.testDescription, func(t *testing.T) {
			defer mockedTracer.Reset()
			request := httptest.NewRequest(http.MethodGet, "https://apis.somecompany.com/endpoint", nil)
			next := func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(test.httpStatus)
			}

			tracer.ServeHTTP(negroni.NewResponseWriter(httptest.NewRecorder()), request, next)

			spans := mockedTracer.FinishedSpans()
			assert.Len(t, spans, 1)
			span := spans[0]

			assert.Equal(t, test.expectedTags, span.Tags())
		})
	}
}

func TestShouldContinueTraceIfAlreadyPresent(t *testing.T) {
	defer mockedTracer.Reset()
	parentSpan := mockedTracer.StartSpan("some-operation").(*mocktracer.MockSpan)
	ext.SpanKindRPCClient.Set(parentSpan)
	request := httptest.NewRequest(http.MethodGet, "https://apis.somecompany.com/endpoint", nil)
	carrier := opentracing.HTTPHeadersCarrier(request.Header)
	// this request now contains a trace initiated by another service/process (e.g. an edge proxy that fronts Hydra)
	require.NoError(t, mockedTracer.Inject(parentSpan.Context(), opentracing.HTTPHeaders, carrier))

	next := func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}

	tracer.ServeHTTP(negroni.NewResponseWriter(httptest.NewRecorder()), request, next)

	spans := mockedTracer.FinishedSpans()
	assert.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, parentSpan.SpanContext.SpanID, span.ParentID)
}

func TestShouldNotTraceHealthEndpoint(t *testing.T) {
	testCases := []struct {
		path            string
		testDescription string
	}{
		{
			path:            "health/ready",
			testDescription: "ready",
		},
		{
			path:            "health/alive",
			testDescription: "alive",
		},
	}
	for _, test := range testCases {
		t.Run(test.testDescription, func(t *testing.T) {
			defer mockedTracer.Reset()
			request := httptest.NewRequest(http.MethodGet, "https://apis.somecompany.com/"+test.path, nil)

			next := func(rw http.ResponseWriter, _ *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}

			tracer.ServeHTTP(negroni.NewResponseWriter(httptest.NewRecorder()), request, next)

			spans := mockedTracer.FinishedSpans()
			assert.Len(t, spans, 0)
		})
	}
}

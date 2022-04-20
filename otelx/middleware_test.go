package otelx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/negroni"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestShouldNotTraceHealthEndpoint(t *testing.T) {
	testCases := []struct {
		path            string
		testDescription string
	}{
		{
			path:            "health/ready",
			testDescription: "health",
		},
		{
			path:            "foo/bar",
			testDescription: "notHealth",
		},
	}
	for _, test := range testCases {
		t.Run(test.testDescription, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))

			req := httptest.NewRequest(http.MethodGet, "https://api.example.com/"+test.path, nil)
			h := NewHandler(negroni.New(), "test op")
			h.ServeHTTP(negroni.NewResponseWriter(httptest.NewRecorder()), req)

			spans := recorder.Ended()
			if strings.Contains(test.path, "health") {
				assert.Len(t, spans, 0)
			} else {
				assert.Len(t, spans, 1)
			}
		})
	}
}

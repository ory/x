package otelx

import (
	"net/http"
	"net/http/httptest"
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
			testDescription: "ready",
		},
		{
			path:            "health/alive",
			testDescription: "alive",
		},
	}
	for _, test := range testCases {
		t.Run(test.testDescription, func(t *testing.T) {
			ime := tracetest.NewInMemoryExporter()
			tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(ime))
			otel.SetTracerProvider(tp)
			defer ime.Reset()

			req := httptest.NewRequest(http.MethodGet, "https://api.example.com/"+test.path, nil)
			h := NewHandler(negroni.New(), "test op")
			h.ServeHTTP(negroni.NewResponseWriter(httptest.NewRecorder()), req)

			spans := ime.GetSpans()
			assert.Len(t, spans, 0)
		})
	}

}

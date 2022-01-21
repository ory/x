package tracing

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTracingRoundTripper(t *testing.T) {
	tracer := mocktracer.New()

	// mock Round Tripper to just return a valid response
	var mockRoundTripper http.RoundTripper = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Status:     http.StatusText(http.StatusNotImplemented),
			StatusCode: http.StatusNotImplemented,
		}, nil
	})

	rt := RoundTripper(tracer, mockRoundTripper)

	parent := tracer.StartSpan("parent")
	ctx := opentracing.ContextWithSpan(context.Background(), parent)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req = req.WithContext(ctx)
	res, err := rt.RoundTrip(req)

	parent.Finish()
	assert.Nil(t, err)
	assert.Equal(t, http.StatusNotImplemented, res.StatusCode)

	// Did this create the expected spans?
	recordedSpans := tracer.FinishedSpans()
	assert.Len(t, recordedSpans, 2)
	assert.Equal(t, "webhook", recordedSpans[0].OperationName)
	assert.Equal(t, "GET", recordedSpans[0].Tag("http.method"))
	assert.Equal(t, "http://example.com/foo", recordedSpans[0].Tag("http.url"))
	assert.Equal(t, uint16(http.StatusNotImplemented), recordedSpans[0].Tag("http.status_code"))

	assert.Equal(t, recordedSpans[1].OperationName, "parent")
}

func TestTracingRoundTripper_error(t *testing.T) {
	tracer := mocktracer.New()

	var mockRoundTripper http.RoundTripper = RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("round trip failed!")
	})

	rt := RoundTripper(tracer, mockRoundTripper)

	parent := tracer.StartSpan("parent")
	ctx := opentracing.ContextWithSpan(context.Background(), parent)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req = req.WithContext(ctx)
	res, err := rt.RoundTrip(req)

	parent.Finish()
	assert.Error(t, err)
	assert.Nil(t, res)

	recordedSpans := tracer.FinishedSpans()

	assert.Len(t, recordedSpans, 2)
	assert.Equal(t, "webhook", recordedSpans[0].OperationName)
	assert.Equal(t, "GET", recordedSpans[0].Tag("http.method"))
	assert.Equal(t, "http://example.com/foo", recordedSpans[0].Tag("http.url"))
	assert.Equal(t, "round trip failed!", recordedSpans[0].Tag("http.error"))
	assert.NotNil(t, recordedSpans[0].FinishTime)
	assert.Equal(t, "parent", recordedSpans[1].OperationName)
}

package tracing_test

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/tracing"

	"go.elastic.co/apm/transport"
)

type zipkinSpanRequest struct {
	Id            string
	TraceId       string
	Timestamp     uint64
	Name          string
	LocalEndpoint struct {
		ServiceName string
	}
	Tags map[string]string
}

type elasticMetadataRequest struct {
	Metadata struct {
		Service struct {
			Name string
		}
	}
}

type elasticSpanRequest struct {
	Transaction struct {
		Name      string
		Id        string
		Timestamp uint64
		TraceId   string
		Type      string
		Context   struct {
			Tags map[string]string
		}
	}
}

func TestZipkinTracer(t *testing.T) {
	done := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)

		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var spans []zipkinSpanRequest
		err = json.Unmarshal(body, &spans)

		assert.NoError(t, err)

		assert.NotEmpty(t, spans[0].Id)
		assert.NotEmpty(t, spans[0].TraceId)
		assert.Equal(t, "testOperation", spans[0].Name)
		assert.Equal(t, "Hydra", spans[0].LocalEndpoint.ServiceName)
		assert.NotNil(t, spans[0].Tags["testTag"])
		assert.Equal(t, "true", spans[0].Tags["testTag"])
	}))
	defer ts.Close()

	tracer := &tracing.Tracer{
		ServiceName: "Hydra",
		ZipkinConfig: &tracing.ZipkinConfig{
			ServerURL: ts.URL,
		},
		Provider: "zipkin",
		Logger:   logrusx.New("Hydra", "1"),
	}
	err := tracer.Setup()
	assert.NoError(t, err)

	span := opentracing.GlobalTracer().StartSpan("testOperation")
	span.SetTag("testTag", true)
	span.Finish()

	select {
	case <-done:
	case <-time.After(time.Millisecond * 1500):
		t.Fatalf("Test server did not receive spans")
	}
}

func TestElastcApmTracer(t *testing.T) {
	done := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)

		switch r.URL.String() {
		case "/config/v1/agents?service.name=ORY+Hydra":
			break
		case "/intake/v2/events":
			body := decodeResponseBody(t, r)
			fmt.Println(string(body))
			data := bytes.Split(body, []byte("\n"))
			assert.GreaterOrEqual(t, len(data), 2)
			var metadata elasticMetadataRequest
			err := json.Unmarshal(data[0], &metadata)
			assert.NoError(t, err)
			assert.Equal(t, "ORY Hydra", metadata.Metadata.Service.Name)

			var spans elasticSpanRequest
			err = json.Unmarshal(data[1], &spans)
			assert.Equal(t, "testOperation", spans.Transaction.Name)
			assert.Equal(t, "custom", spans.Transaction.Type)
			assert.Equal(t, "true", spans.Transaction.Context.Tags["testTag"])

			break
		default:
			t.Fatalf("Unknown request:" + r.URL.String())
		}

	}))
	defer ts.Close()

	os.Setenv("ELASTIC_APM_SERVER_URL", ts.URL)
	//Reset env vars in APM Library
	transport.InitDefault()

	tracer := &tracing.Tracer{
		ServiceName: "ORY Hydra",
		Provider:    "elastic-apm",
		Logger:      logrusx.New("ORY Hydra", "1"),
	}

	err := tracer.Setup()
	assert.NoError(t, err)

	span := opentracing.GlobalTracer().StartSpan("testOperation")
	span.SetTag("testTag", true)
	span.Finish()

	select {
	case <-done:
	case <-time.After(time.Millisecond * 1500):
		t.Fatalf("Test server did not receive spans")
	}
}

func decodeResponseBody(t *testing.T, r *http.Request) []byte {
	var reader io.ReadCloser
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		reader, err = gzip.NewReader(r.Body)
		if err != nil {
			t.Fatal(err)
		}
	case "deflate":
		var err error
		reader, err = zlib.NewReader(r.Body)
		if err != nil {
			t.Fatal(err)
		}

	default:
		reader = r.Body
	}
	respBody, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
		//return []byte{}
	}
	reader.Close()

	return respBody
}

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
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/tracing"

	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

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
		assert.Equal(t, "testoperation", spans[0].Name)
		assert.Equal(t, "ory x", spans[0].LocalEndpoint.ServiceName)
		assert.NotNil(t, spans[0].Tags["testTag"])
		assert.Equal(t, "true", spans[0].Tags["testTag"])
	}))
	defer ts.Close()

	_, err := tracing.New(logrusx.New("ory/x", "1"), &tracing.Config{
		ServiceName: "ORY X",
		Provider:    "zipkin",
		Providers: &tracing.ProvidersConfig{
			Zipkin: &tracing.ZipkinConfig{
				ServerURL: ts.URL,
			},
		},
	})
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
	done := make(chan struct{}, 2)
	defer close(done)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Got connection!")
		done <- struct{}{}

		switch r.URL.String() {
		case "/config/v1/agents?service.name=ORY+X":
			break
		case "/intake/v2/events":
			body := decodeResponseBody(t, r)
			fmt.Println(string(body))
			data := bytes.Split(body, []byte("\n"))
			assert.GreaterOrEqual(t, len(data), 2)
			var metadata elasticMetadataRequest
			err := json.Unmarshal(data[0], &metadata)
			assert.NoError(t, err)
			assert.Equal(t, "ORY X", metadata.Metadata.Service.Name)

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

	require.NoError(t, os.Setenv("ELASTIC_APM_SERVER_URL", ts.URL))
	// Reset env vars in APM Library
	_, err := transport.InitDefault()
	require.NoError(t, err)

	_, err = tracing.New(logrusx.New("ory/x", "1"), &tracing.Config{
		ServiceName: "ORY X",
		Provider:    "elastic-apm",
		Providers: &tracing.ProvidersConfig{
			Zipkin: &tracing.ZipkinConfig{
				ServerURL: ts.URL,
			},
		},
	})
	require.NoError(t, err)

	span := opentracing.GlobalTracer().StartSpan("testOperation")
	span.SetTag("testTag", true)
	span.Finish()

	for i := 0; i < 2; i++ {
		select {
		case _, ok := <-done:
			if !ok {
				return
			}
		case <-time.After(time.Millisecond * 1500):
			t.Fatalf("Test server did not receive spans")
			return
		}
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
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return respBody
}

func TestInstanaTracer(t *testing.T) {
	done := make(chan struct{})

	type discoveryRequest struct {
		PID   int      `json:"pid"`
		Name  string   `json:"name"`
		Args  []string `json:"args"`
		Fd    string   `json:"fd"`
		Inode string   `json:"inode"`
	}

	type discoveryResponse struct {
		Pid     uint32 `json:"pid"`
		HostID  string `json:"agentUuid"`
		Secrets struct {
			Matcher string   `json:"matcher"`
			List    []string `json:"list"`
		} `json:"secrets"`
		ExtraHTTPHeaders []string `json:"extraHeaders"`
	}

	type traceRequest struct {
		Timestamp uint64 `json:"ts"`
		Data      struct {
			Service string `json:"service"`
			Sdk     struct {
				Name   string `json:"name"`
				Type   string `json:"type"`
				Custom struct {
					Baggage map[string]interface{}            `json:"baggage"`
					Logs    map[uint64]map[string]interface{} `json:"logs"`
					Tags    map[string]interface{}            `json:"tags"`
				} `json:"custom"`
			} `json:"sdk"`
		} `json:"data"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			t.Log("Got Agent check request")

			w.Header().Set("Server", "Instana Agent")
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/com.instana.plugin.golang.discovery" {
			t.Log("Got Agent discovery request")

			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)

			var dReq discoveryRequest
			assert.NoError(t, json.Unmarshal(body, &dReq))

			agentResponse := discoveryResponse{
				Pid:    1,
				HostID: "1",
			}
			resp, err := json.Marshal(&agentResponse)
			assert.NoError(t, err)
			w.Header().Set("Server", "Instana Agent")
			w.Write(resp)
			return
		}

		if strings.Contains(r.URL.Path, "/com.instana.plugin.golang/traces.") {
			t.Log("Got trace request")

			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)

			var req []traceRequest
			assert.NoError(t, json.Unmarshal(body, &req))

			assert.Equal(t, "ORY X", req[0].Data.Service)
			assert.Equal(t, "testOperation", req[0].Data.Sdk.Name)
			assert.Equal(t, true, req[0].Data.Sdk.Custom.Tags["testTag"])
			assert.Equal(t, "biValue", req[0].Data.Sdk.Custom.Baggage["testBi"])
			//assert.Equal(t, "testValue", req[0].Data.Sdk.Custom.Logs[req[0].Timestamp]["testKey"])

			w.Header().Set("Server", "Instana Agent")
			w.WriteHeader(http.StatusOK)

			close(done)
			return
		}
	}))
	defer ts.Close()

	agentUrl, err := url.Parse(ts.URL)
	require.NoError(t, err)

	require.NoError(t, os.Setenv("INSTANA_AGENT_HOST", agentUrl.Hostname()))
	require.NoError(t, os.Setenv("INSTANA_AGENT_PORT", agentUrl.Port()))

	_, err = tracing.New(logrusx.New("ory/x", "1"), &tracing.Config{
		ServiceName: "ORY X",
		Provider:    "instana",
	})
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	span := opentracing.GlobalTracer().StartSpan("testOperation")
	span.SetTag("testTag", true)
	span.LogKV("testKey", "testValue")
	span.SetBaggageItem("testBi", "biValue")
	span.Finish()

	select {
	case <-done:
	case <-time.After(time.Second * 3):
		t.Fatalf("Test server did not receive spans")
	}
}

func TestOtlpTracer(t *testing.T) {
	done := make(chan struct{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeResponseBody(t, r)

		var res coltracepb.ExportTraceServiceRequest
		err := proto.Unmarshal(body, &res)
		require.NoError(t, err, "must be able to unmarshal traces")
		receivedSpan := res.ResourceSpans[0].InstrumentationLibrarySpans[0].Spans[0]
		assert.Equal(t, "testOperation", receivedSpan.GetName())
		attributes := receivedSpan.GetAttributes()
		assert.Equal(t, "testTag", attributes[0].GetKey())

		close(done)
	}))
	defer ts.Close()

	require.NoError(t, os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", ts.URL))

	_, err := tracing.New(logrusx.New("ory/x", "1"), &tracing.Config{
		ServiceName: "ORY X",
		Provider:    "otel",
	})
	assert.NoError(t, err)

	span := opentracing.GlobalTracer().StartSpan("testOperation")
	span.SetTag("testTag", true)
	span.Finish()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("Test server did not receive spans")
	}
}

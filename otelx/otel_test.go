package otelx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/instana/testify/assert"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/ory/x/logrusx"
)

const testTracingComponent = "github.com/ory/x/otelx"

func TestJaegerTracer(t *testing.T) {
	done := make(chan struct{})
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	errs := errgroup.Group{}
	errs.Go(func() error {
		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}

		t.Logf("Starting test UDP server for Jaeger spans on %s", udpAddr.String())

		srv, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			return err
		}

		for {
			buf := make([]byte, 2048)
			_, conn, err := srv.ReadFromUDP(buf)
			if err != nil {
				return err
			}

			if conn == nil {
				continue
			}
			if len(buf) != 0 {
				t.Log("recieved span!")
				done <- struct{}{}
			}
			break
		}
		return nil
	})

	ot, err := New(testTracingComponent, logrusx.New("ory/x", "1"), &Config{
		ServiceName: "Ory X",
		Provider:    "jaeger",
		Providers: ProvidersConfig{
			Jaeger: JaegerConfig{
				LocalAgentAddress: addr,
			},
		},
	})
	require.NoError(t, err)

	trc := ot.Tracer()
	_, span := trc.Start(context.Background(), "testSpan")
	span.SetAttributes(attribute.Bool("testAttribute", true))
	span.End()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Log("expected to receive span, but did not receive any")
		t.Fail()
	}
	require.NoError(t, errs.Wait())
}

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

func TestZipkinTracer(t *testing.T) {
	done := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var spans []zipkinSpanRequest
		err = json.Unmarshal(body, &spans)

		assert.NoError(t, err)

		assert.NotEmpty(t, spans[0].Id)
		assert.NotEmpty(t, spans[0].TraceId)
		assert.Equal(t, "testspan", spans[0].Name)
		assert.Equal(t, "ory x", spans[0].LocalEndpoint.ServiceName)
		assert.NotNil(t, spans[0].Tags["testTag"])
		assert.Equal(t, "true", spans[0].Tags["testTag"])
	}))
	defer ts.Close()

	zt, err := New(testTracingComponent, logrusx.New("ory/x", "1"), &Config{
		ServiceName: "Ory X",
		Provider:    "zipkin",
		Providers: ProvidersConfig{
			Zipkin: ZipkinConfig{
				ServerURL: ts.URL,
				Sampling: ZipkinSampling{
					SamplingRatio: 1,
				},
			},
		},
	})
	assert.NoError(t, err)

	trc := zt.Tracer()
	_, span := trc.Start(context.Background(), "testspan")
	span.SetAttributes(attribute.Bool("testTag", true))
	span.End()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("Test server did not receive spans")
	}
}

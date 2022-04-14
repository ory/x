package otelx

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ory/x/logrusx"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"
)

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

	ot, err := New("github.com/ory/x/otelx", logrusx.New("ory/x", "1"), &Config{
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

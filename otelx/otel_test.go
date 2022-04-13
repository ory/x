package otelx

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/ory/x/logrusx"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestJaegerTracer(t *testing.T) {
	host := "127.0.0.1"
	port := "6831"
	done := make(chan struct{})
	go func(addr string) {
		t.Log("Starting test UDP server for Jaeger spans")
		udpAddr, _ := net.ResolveUDPAddr("udp", addr)
		srv, _ := net.ListenUDP("udp", udpAddr)
		for {
			buf := make([]byte, 2048)
			_, conn, _ := srv.ReadFromUDP(buf)
			if conn == nil {
				continue
			}
			if len(buf) != 0 {
				t.Log("recieved span!")
				done <- struct{}{}
			}
			break
		}
	}(fmt.Sprintf("%s:%s", host, port))

	ot, err := New("github.com/ory/x/otelx", logrusx.New("ory/x", "1"), &Config{
		ServiceName: "Ory X",
		Provider:    "jaeger",
		Providers: ProvidersConfig{
			Jaeger: JaegerConfig{
				LocalAgentAddress: "127.0.0.1:6831",
			},
		},
	})
	require.NoError(t, err)

	trc := ot.Tracer()
	_, span := trc.Start(context.Background(), "testSpan")
	span.SetAttributes(attribute.Bool("testAttribute", true))
	span.End()

	<-done
}

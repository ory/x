// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetupOTLPRequestPath(t *testing.T) {
	for _, tc := range []struct {
		name      string
		serverURL func(base, host string) string
		wantPath  string
	}{
		{
			name: "host and port",
			serverURL: func(_, host string) string {
				return host
			},
			wantPath: "/v1/traces",
		},
		{
			name: "grafana cloud style url",
			serverURL: func(base, _ string) string {
				return base + "/otlp"
			},
			wantPath: "/otlp/v1/traces",
		},
		{
			name: "full traces url",
			serverURL: func(base, _ string) string {
				return base + "/v1/traces"
			},
			wantPath: "/v1/traces",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prevTP := otel.GetTracerProvider()
			prevProp := otel.GetTextMapPropagator()
			t.Cleanup(func() {
				otel.SetTracerProvider(prevTP)
				otel.SetTextMapPropagator(prevProp)
			})

			got := make(chan string, 1)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case got <- r.URL.Path:
				default:
				}
			}))
			t.Cleanup(ts.Close)

			tsu, err := url.Parse(ts.URL)
			require.NoError(t, err)

			_, err = SetupOTLP(nil, "test", &Config{
				ServiceName: "ORY X",
				Provider:    "otel",
				Providers: ProvidersConfig{
					OTLP: OTLPConfig{
						ServerURL: tc.serverURL(ts.URL, tsu.Host),
						Insecure:  true,
						Sampling: OTLPSampling{
							SamplingRatio: 1,
						},
					},
				},
			})
			require.NoError(t, err)

			tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
			require.True(t, ok)
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = tp.Shutdown(ctx)
			})

			tracer := tp.Tracer("test")
			_, span := tracer.Start(context.Background(), "testSpan")
			span.SetAttributes(attribute.Bool("testAttribute", true))
			span.End()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, tp.ForceFlush(ctx))

			select {
			case path := <-got:
				assert.Equal(t, tc.wantPath, path)
			case <-ctx.Done():
				t.Fatalf("collector did not receive a span; want path %s", tc.wantPath)
			}
		})
	}
}

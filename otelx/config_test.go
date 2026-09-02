// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
)

func TestOTLPServerURLSchema(t *testing.T) {
	c := jsonschema.NewCompiler()
	require.NoError(t, AddConfigSchema(c))

	schema, err := c.Compile(context.Background(), ConfigSchemaID)
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "host and port", url: "localhost:4318"},
		{name: "ipv4 and port", url: "127.0.0.1:4318"},
		{name: "grafana cloud url", url: "https://otlp-gateway-prod-gb-south-0.grafana.net/otlp"},
		{name: "http url with path", url: "http://collector.example.com:4318/otlp"},
		{name: "https url without path", url: "https://collector.example.com:4318"},
		{name: "bare hostname", url: "localhost", wantErr: true},
		{name: "path without scheme", url: "collector.example.com/otlp", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := fmt.Sprintf(`{"provider":"otel","providers":{"otlp":{"server_url":%q}}}`, tc.url)
			err := schema.Validate(bytes.NewBufferString(doc))
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

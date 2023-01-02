// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package otelx

import "go.opentelemetry.io/otel/attribute"

func StringAttrs(attrs map[string]string) []attribute.KeyValue {
	s := []attribute.KeyValue{}
	for k, v := range attrs {
		s = append(s, attribute.String(k, v))
	}
	return s
}

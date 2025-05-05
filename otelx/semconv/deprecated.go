// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"context"

	otelattr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func NewDeprecatedFeatureUsedEvent(ctx context.Context, deprecatedCodeFeatureID string, attrs ...otelattr.KeyValue) (string, trace.EventOption) {
	return DeprecatedFeatureUsed.String(),
		trace.WithAttributes(
			append(
				append(
					attrs,
					AttributesFromContext(ctx)...,
				),
				AttrDeprecatedFeatureID(deprecatedCodeFeatureID),
			)...,
		)
}

const (
	AttributeKeyDeprecatedCodePathIDAttributeKey AttributeKey = "DeprecatedFeatureID"
	DeprecatedFeatureUsed                        Event        = "DeprecatedFeatureUsed"
)

func AttrDeprecatedFeatureID(id string) otelattr.KeyValue {
	return otelattr.String(AttributeKeyDeprecatedCodePathIDAttributeKey.String(), id)
}

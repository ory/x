// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

// Package semconv contains OpenTelemetry semantic convention constants.
package semconv

import (
	"github.com/gofrs/uuid"
	otelattr "go.opentelemetry.io/otel/attribute"
)

type Event string

func (e Event) String() string {
	return string(e)
}

type AttributeKey string

func (a AttributeKey) String() string {
	return string(a)
}

const (
	AttributeKeyIdentityID AttributeKey = "IdentityID"
	AttributeKeyNID        AttributeKey = "ProjectID"
	AttributeKeyClientIP   AttributeKey = "ClientIP"
)

func AttrIdentityID(val uuid.UUID) otelattr.KeyValue {
	return otelattr.String(AttributeKeyIdentityID.String(), val.String())
}

func AttrNID(val uuid.UUID) otelattr.KeyValue {
	return otelattr.String(AttributeKeyNID.String(), val.String())
}

func AttrClientIP(val string) otelattr.KeyValue {
	return otelattr.String(AttributeKeyClientIP.String(), val)
}

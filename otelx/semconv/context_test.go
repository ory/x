package semconv

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestAttributesFromContext(t *testing.T) {
	ctx := context.Background()
	assert.Len(t, AttributesFromContext(ctx), 0)

	nid := uuid.Must(uuid.NewV4())
	ctx = ContextWithAttributes(ctx, AttrNID(nid))
	assert.Len(t, AttributesFromContext(ctx), 1)

	uid1, uid2 := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	ctx = ContextWithAttributes(ctx, AttrIdentityID(uid1), AttrClientIP("127.0.0.1"), AttrIdentityID(uid2))
	attrs := AttributesFromContext(ctx)
	assert.Len(t, attrs, 3, "should deduplicate")
	assert.Equal(t, []attribute.KeyValue{
		attribute.String(attributeKeyNID.String(), nid.String()),
		attribute.String(attributeKeyClientIP.String(), "127.0.0.1"),
		attribute.String(attributeKeyIdentityID.String(), uid2.String()),
	}, attrs, "last duplicate attribute wins")
}

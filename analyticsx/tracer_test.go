package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"gocloud.dev/pubsub"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/otelx/semconv"

	_ "gocloud.dev/pubsub/mempubsub"
)

func TestTracer(t *testing.T) {
	reachedHandler := make(chan struct{})
	var m Event

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	topic, err := pubsub.OpenTopic(ctx, "mem://test-topic")
	require.NoError(t, err)
	t.Cleanup(func() { _ = topic.Shutdown(context.Background()) })

	subscription, err := pubsub.OpenSubscription(ctx, "mem://test-topic")
	require.NoError(t, err)
	t.Cleanup(func() { _ = subscription.Shutdown(context.Background()) })

	go func() {
		for {
			msg, err := subscription.Receive(ctx)
			require.NoError(t, err)
			msg.Ack()
			require.NoError(t, proto.Unmarshal(msg.Body, &m))
			reachedHandler <- struct{}{}
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	}()

	noop := trace.NewNoopTracerProvider().Tracer("test-tracer")
	tracer := WrapTracer(noop, topic, logrusx.New("test-logger", ""))

	// root span
	ctx, span := tracer.Start(ctx, "outer span")
	span.AddEvent("IdentityCreated", trace.WithAttributes(
		attribute.String(semconv.AttributeKeyIdentityID.String(), "test-distinct-id"),
		attribute.String(semconv.AttributeKeyNID.String(), "test-nid"),
	))
	select {
	case <-time.After(2000 * time.Millisecond):
		t.Fatal("execution didn't reach handler")
	case <-reachedHandler: // OK
	}

	assert.Equal(t, "test-distinct-id", m.EventAttributes[semconv.AttributeKeyIdentityID.String()])
	assert.Equal(t, "IdentityCreated", m.Name)
	assert.Equal(t, "test-nid", m.ProjectId)

	// child span
	_, span = tracer.Start(ctx, "inner span")
	span.AddEvent("SessionIssued", trace.WithAttributes(
		attribute.String(semconv.AttributeKeyIdentityID.String(), "test-distinct-id-2"),
		attribute.String(semconv.AttributeKeyNID.String(), "test-nid-2"),
	))
	select {
	case <-time.After(2000 * time.Millisecond):
		t.Fatal("execution didn't reach handler")
	case <-reachedHandler: // OK
	}

	assert.Equal(t, "test-distinct-id-2", m.EventAttributes[semconv.AttributeKeyIdentityID.String()])
	assert.Equal(t, "SessionIssued", m.Name)
	assert.Equal(t, "test-nid-2", m.ProjectId)
}

func TestNoPanic(t *testing.T) {
	noop := trace.NewNoopTracerProvider().Tracer("test-tracer")
	tracer := WrapTracer(noop, nil, nil)
	assert.NotNil(t, tracer)
	_, span := tracer.Start(context.Background(), "")
	assert.NotPanics(t, func() { span.AddEvent("test event nil analytics") })

	assert.Equal(t, span.TracerProvider().Tracer(""), tracer)
}

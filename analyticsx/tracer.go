//go:generate buf generate .

package analytics

import (
	"context"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"gocloud.dev/pubsub/gcppubsub"
	"golang.org/x/oauth2/google"

	"github.com/ory-corp/cloud/cloudlib"
	"github.com/ory/x/otelx"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gocloud.dev/pubsub"
	"google.golang.org/protobuf/proto"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/otelx/semconv"
)

func OpenTopicForPublishing(ctx context.Context, jsonCredentials, topicName string) (*pubsub.Topic, error) {
	creds, err := google.CredentialsFromJSON(ctx, []byte(jsonCredentials), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	conn, _, err := gcppubsub.Dial(ctx, creds.TokenSource)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	pubClient, err := gcppubsub.PublisherClient(ctx, conn)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return gcppubsub.OpenTopicByPath(pubClient, topicName, nil)
}

// WrapTracer wraps any OpenTelemetry-compatible tracer with functionality to
// send span events and their attributes
// (https://opentelemetry.io/docs/concepts/signals/traces/#span-events) to our
// analytics backend.
//
// The returned tracer functions identically to the wrapped tracer, but performs
// additional actions when the AddEvents method is called on its spans.
func WrapTracer(t trace.Tracer, topic *pubsub.Topic, l *logrusx.Logger) trace.Tracer {
	if t == nil {
		return nil
	}

	if l == nil {
		return &tracer{t, topic, nil}
	}
	return &tracer{t, topic, l.WithField("component", "tracer")}
}

// WrapTracerWithPubSubFromEnv wraps any OpenTelemetry-compatible tracer with a tracer that
// sends events to a pubsub topic. The pubsub topic and credentials are read from the environment.
//
// If no topic or credentials are found in the environment, the original tracer is returned unmodified.
func WrapTracerWithPubSubFromEnv(ctx context.Context, tracer *otelx.Tracer, l *logrusx.Logger) *otelx.Tracer {
	l = l.WithField("component", "tracer")
	if tracer == nil {
		l.Info("analytics disabled because no tracer was loaded")
		return nil
	}

	credentials, ok := os.LookupEnv("PROJECT_METRICS_PUBLISHER_SECRET")
	if !ok {
		l.Info("analytics disabled because no API key was found in the environment")
		return tracer
	}

	topicName, ok := os.LookupEnv("PROJECT_METRICS_PUBSUB_TOPIC")
	if !ok {
		l.Info("analytics disabled because no topic name was found in the environment")
		return tracer
	}

	topic, err := OpenTopicForPublishing(ctx, credentials, topicName)
	if err != nil {
		l.WithError(err).Warn("failed to open pubsub topic, tracer unmodified")
		return tracer
	}

	wrapped := WrapTracer(tracer.Tracer(), topic, l)
	l.Info("enabled GCP pubsub analytics via OpenTelemetry")

	return tracer.WithOTLP(wrapped)
}

type tracer struct {
	trace.Tracer
	topic *pubsub.Topic
	l     *logrusx.Logger
}

// Start implements trace.Tracer.
func (t *tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t == nil || t.Tracer == nil {
		return ctx, nil
	}
	if span, isWrapped := trace.SpanFromContext(ctx).(*span); isWrapped {
		// If we created the span from the context ourselves: pass the naked
		// span to our wrapped tracer.
		ctx = trace.ContextWithSpan(ctx, span.Span)
	}
	_, inner := t.Tracer.Start(ctx, spanName, opts...)
	outer := &span{inner, t}
	return trace.ContextWithSpan(ctx, outer), outer
}

type span struct {
	trace.Span
	t *tracer
}

func lookupAttribute(attrs []attribute.KeyValue, key semconv.AttributeKey) string {
	for i := range attrs {
		if !attrs[i].Valid() {
			continue
		}
		if string(attrs[i].Key) == key.String() {
			return attrs[i].Value.AsString()
		}
	}
	return ""
}

// AddEvent implements part of trace.Span. We send the event and it's attributes
// to our analytics backend. The event is also still reported to the wrapped
// tracer unchanged.
func (s *span) AddEvent(name string, options ...trace.EventOption) {
	if s == nil {
		return
	}
	s.Span.AddEvent(name, options...)

	if s.t.topic == nil {
		return
	}
	c := trace.NewEventConfig(options...)
	attrs := c.Attributes()

	projectID := lookupAttribute(attrs, semconv.AttributeKeyNID)
	attributes := make(map[string]string)
	for _, attr := range attrs {
		if !attr.Valid() || string(attr.Key) == string(semconv.AttributeKeyNID) {
			continue
		}
		attributes[string(attr.Key)] = attr.Value.AsString()
	}

	// Add a flag to indicate if the project is a test project
	// If the UUID is not valid, we assume it's not a test project
	if id, err := uuid.FromString(projectID); err == nil && cloudlib.IsTestProject(id) {
		attributes["IsTestProject"] = "true"
	}

	event := &Event{
		Name:            name,
		ProjectId:       projectID,
		Version:         2,
		Timestamp:       time.Now().UTC().Format("2006-01-02 15:04:05"),
		EventAttributes: attributes,
		Source:          Source_SOURCE_PUBLIC_NETWORK,
	}

	body, err := proto.Marshal(event)
	if err != nil {
		return
	}
	err = s.t.topic.Send(context.Background(), &pubsub.Message{Body: body})
	if err != nil {
		return
	}
	s.t.l.Debugf("sent event to pubsub: %+v", event)
}

// TracerProvider implements part of trace.Span.
func (s *span) TracerProvider() trace.TracerProvider {
	if s == nil {
		return nil
	}
	return provider{s.t}
}

type provider struct {
	t *tracer
}

var _ trace.TracerProvider = provider{}

// Tracer implements trace.TracerProvider.
func (p provider) Tracer(_ string, _ ...trace.TracerOption) trace.Tracer {
	return p.t
}

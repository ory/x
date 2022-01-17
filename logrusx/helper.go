package logrusx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gobuffalo/pop/v6/logging"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel/propagation"

	"github.com/ory/x/errorsx"
)

type Logger struct {
	*logrus.Entry
	leakSensitive bool
	opts          []Option
	name          string
	version       string
}

var opts = otelhttptrace.WithPropagators(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

func (l *Logger) LeakSensitiveData() bool {
	return l.leakSensitive
}

func (l *Logger) Logrus() *logrus.Logger {
	return l.Entry.Logger
}

func (l *Logger) NewEntry() *Logger {
	ll := *l
	ll.Entry = logrus.NewEntry(l.Logger)
	return &ll
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	ll := *l
	ll.Entry = l.Logger.WithContext(ctx)
	return &ll
}

func (l *Logger) HTTPHeadersRedacted(h http.Header) map[string]interface{} {
	headers := map[string]interface{}{}
	if cookie := l.maybeRedact(h.Get("Cookie")); cookie != nil {
		headers["cookie"] = cookie
	}

	if auth := l.maybeRedact(h.Get("Authorization")); auth != nil {
		headers["authorization"] = auth
	}

	for key := range h {
		if strings.ToLower(key) == "cookie" ||
			strings.ToLower(key) == "authorization" {
			continue
		}
		headers[strings.ToLower(key)] = h.Get(key)
	}

	return headers
}

func (l *Logger) WithRequest(r *http.Request) *Logger {
	headers := l.HTTPHeadersRedacted(r.Header)
	if ua := r.UserAgent(); len(ua) > 0 {
		headers["user-agent"] = ua
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	ll := l.WithField("http_request", map[string]interface{}{
		"remote":  r.RemoteAddr,
		"method":  r.Method,
		"path":    r.URL.EscapedPath(),
		"query":   l.maybeRedact(r.URL.RawQuery),
		"scheme":  scheme,
		"host":    r.Host,
		"headers": headers,
	})

	if _, _, spanCtx := otelhttptrace.Extract(r.Context(), r, opts); spanCtx.IsValid() {
		traces := map[string]string{}
		if spanCtx.HasTraceID() {
			traces["trace_id"] = spanCtx.TraceID().String()
		}
		if spanCtx.HasSpanID() {
			traces["span_id"] = spanCtx.SpanID().String()
		}
		ll = ll.WithField("otel", traces)
	}

	return ll
}

func (l *Logger) WithFields(f logrus.Fields) *Logger {
	ll := *l
	ll.Entry = l.Entry.WithFields(f)
	return &ll
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	ll := *l
	ll.Entry = l.Entry.WithField(key, value)
	return &ll
}

func (l *Logger) maybeRedact(value interface{}) interface{} {
	if fmt.Sprintf("%v", value) == "" || value == nil {
		return nil
	}
	if !l.leakSensitive {
		return `Value is sensitive and has been redacted. To see the value set config key "log.leak_sensitive_values = true" or environment variable "LOG_LEAK_SENSITIVE_VALUES=true".`
	}
	return value
}

func (l *Logger) WithSensitiveField(key string, value interface{}) *Logger {
	return l.WithField(key, l.maybeRedact(value))
}

func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}

	ctx := map[string]interface{}{"message": err.Error()}
	if l.Entry.Logger.IsLevelEnabled(logrus.DebugLevel) {
		if e, ok := err.(errorsx.StackTracer); ok {
			ctx["stack_trace"] = fmt.Sprintf("%+v", e.StackTrace())
		} else {
			ctx["stack_trace"] = fmt.Sprintf("stack trace could not be recovered from error type %s", reflect.TypeOf(err))
		}
	}
	if c := errorsx.ReasonCarrier(nil); errors.As(err, &c) {
		ctx["reason"] = c.Reason()
	}
	if c := errorsx.RequestIDCarrier(nil); errors.As(err, &c) && c.RequestID() != "" {
		ctx["request_id"] = c.RequestID()
	}
	if c := errorsx.DetailsCarrier(nil); errors.As(err, &c) && c.Details() != nil {
		ctx["details"] = c.Details()
	}
	if c := errorsx.StatusCarrier(nil); errors.As(err, &c) && c.Status() != "" {
		ctx["status"] = c.Status()
	}
	if c := errorsx.StatusCodeCarrier(nil); errors.As(err, &c) && c.StatusCode() != 0 {
		ctx["status_code"] = c.StatusCode()
	}
	if c := errorsx.DebugCarrier(nil); errors.As(err, &c) {
		ctx["debug"] = c.Debug()
	}

	return l.WithField("error", ctx)
}

var popLevelTranslations = map[logging.Level]logrus.Level{
	// logging.SQL:   logrus.TraceLevel, we never want to log SQL statements, see https://github.com/ory/keto/issues/454
	logging.Debug: logrus.DebugLevel,
	logging.Info:  logrus.InfoLevel,
	logging.Warn:  logrus.WarnLevel,
	logging.Error: logrus.ErrorLevel,
}

func (l *Logger) PopLogger(lvl logging.Level, s string, args ...interface{}) {
	level, ok := popLevelTranslations[lvl]
	if ok {
		l.WithField("source", "pop").Logf(level, s, args...)
	}
}

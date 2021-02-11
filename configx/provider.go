package configx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ory/x/logrusx"

	"github.com/ory/x/jsonschemax"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/x/watcherx"

	"github.com/inhies/go-bytesize"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/pflag"

	"github.com/ory/x/stringsx"
	"github.com/ory/x/tracing"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/pkg/errors"
	"github.com/rs/cors"
)

type tuple struct {
	Key   string
	Value interface{}
}

type Provider struct {
	l sync.Mutex
	*koanf.Koanf
	immutables []string

	originalContext context.Context
	cancelFork      context.CancelFunc

	schema                   []byte
	flags                    *pflag.FlagSet
	validator                *jsonschema.Schema
	onChanges                []func(watcherx.Event, error)
	onValidationError        func(k *koanf.Koanf, err error)
	excludeFieldsFromTracing []string
	tracer                   *tracing.Tracer
	forcedValues             []tuple
	baseValues               []tuple
	files                    []string
	skipValidation           bool
	logger                   *logrusx.Logger
}

const (
	FlagConfig = "config"
	Delimiter  = "."
)

// RegisterConfigFlag registers the "--config" flag on pflag.FlagSet.
func RegisterConfigFlag(flags *pflag.FlagSet, fallback []string) {
	flags.StringSliceP(FlagConfig, "c", fallback, "Config files to load, overwriting in the order specified.")
}

// New creates a new provider instance or errors.
// Configuration values are loaded in the following order:
//
// 1. Defaults from the JSON Schema
// 2. Config files (yaml, yml, toml, json)
// 3. Command line flags
// 4. Environment variables
func New(schema []byte, modifiers ...OptionModifier) (*Provider, error) {
	schemaID, comp, err := newCompiler(schema)
	if err != nil {
		return nil, err
	}

	validator, err := comp.Compile(schemaID)
	if err != nil {
		return nil, err
	}

	l := logrus.New()
	l.Out = ioutil.Discard

	p := &Provider{
		originalContext:          context.Background(),
		schema:                   schema,
		validator:                validator,
		onValidationError:        func(k *koanf.Koanf, err error) {},
		excludeFieldsFromTracing: []string{"dsn", "secret", "password", "key"},
		logger:                   logrusx.New("discarding config logger", "", logrusx.UseLogger(l)),
	}

	for _, m := range modifiers {
		m(p)
	}

	k, _, cancelFork, err := p.forkKoanf()
	if err != nil {
		return nil, err
	}

	p.replaceKoanf(k, cancelFork)
	return p, nil
}

func (p *Provider) replaceKoanf(k *koanf.Koanf, cancelFork context.CancelFunc) {
	p.l.Lock()
	defer p.l.Unlock()
	if p.cancelFork != nil {
		p.cancelFork()
	}
	p.Koanf = k
	p.cancelFork = cancelFork
}

func (p *Provider) validate(k *koanf.Koanf) error {
	if p.skipValidation {
		return nil
	}

	out, err := k.Marshal(json.Parser())
	if err != nil {
		return errors.WithStack(err)
	}
	if err := p.validator.Validate(bytes.NewReader(out)); err != nil {
		p.onValidationError(k, err)
		return err
	}

	return nil
}

func (p *Provider) forkKoanf() (*koanf.Koanf, context.Context, context.CancelFunc, error) {
	fork, cancel := context.WithCancel(p.originalContext)
	span, fork := p.startSpan(fork, LoadSpanOpName)
	defer span.Finish()

	k := koanf.New(Delimiter)
	dp, err := NewKoanfSchemaDefaults(p.schema)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	ep, err := NewKoanfEnv("", p.schema)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	// Load defaults
	if err := k.Load(dp, nil); err != nil {
		cancel()
		return nil, nil, nil, err
	}

	for _, t := range p.baseValues {
		if err := k.Load(NewKoanfConfmap([]tuple{t}), nil); err != nil {
			cancel()
			return nil, nil, nil, err
		}
	}

	var paths []string
	if p.flags != nil {
		p, _ := p.flags.GetStringSlice(FlagConfig)
		paths = append(paths, p...)
	}

	if err := p.addAndWatchConfigFiles(fork, append(p.files, paths...), k); err != nil {
		cancel()
		return nil, nil, nil, err
	}

	if p.flags != nil {
		if err := k.Load(posflag.Provider(p.flags, ".", k), nil); err != nil {
			cancel()
			return nil, nil, nil, err
		}
	}

	if err := k.Load(ep, nil); err != nil {
		cancel()
		return nil, nil, nil, err
	}

	// Workaround for https://github.com/knadh/koanf/pull/47
	for _, t := range p.forcedValues {
		if err := k.Load(NewKoanfConfmap([]tuple{t}), nil); err != nil {
			cancel()
			return nil, nil, nil, err
		}
	}

	if err := p.validate(k); err != nil {
		cancel()
		return nil, nil, nil, err
	}

	p.traceConfig(fork, k, LoadSpanOpName)
	return k, fork, cancel, nil
}

// TraceSnapshot will send the configuration to the tracer.
func (p *Provider) SetTracer(ctx context.Context, t *tracing.Tracer) {
	p.tracer = t
	p.traceConfig(ctx, p.Koanf, SnapshotSpanOpName)
}

func (p *Provider) startSpan(ctx context.Context, opName string) (opentracing.Span, context.Context) {
	tracer := opentracing.GlobalTracer()
	if p.tracer != nil && p.tracer.Tracer() != nil {
		tracer = p.tracer.Tracer()
	}
	return opentracing.StartSpanFromContextWithTracer(ctx, tracer, opName)
}

func (p *Provider) traceConfig(ctx context.Context, k *koanf.Koanf, opName string) {
	span, ctx := p.startSpan(ctx, opName)
	defer span.Finish()

	span.SetTag("component", "github.com/ory/x/configx")

	fields := make([]log.Field, 0, len(k.Keys()))
	for _, key := range k.Keys() {
		var redact bool
		for _, e := range p.excludeFieldsFromTracing {
			if strings.Contains(key, e) {
				redact = true
			}
		}

		if redact {
			fields = append(fields, log.Object(key, "[redacted]"))
		} else {
			fields = append(fields, log.Object(key, k.Get(key)))
		}
	}

	span.LogFields(fields...)
}

func (p *Provider) runOnChanges(e watcherx.Event, err error) {
	for k := range p.onChanges {
		p.onChanges[k](e, err)
	}
}

func (p *Provider) addAndWatchConfigFiles(ctx context.Context, paths []string, k *koanf.Koanf) error {
	p.logger.WithField("files", paths).Debug("Adding config files.")

	watchForFileChanges := func(c watcherx.EventChannel) {
		// Channel is closed automatically on ctx.Done() because of fp.WatchChannel()
		for e := range c {
			switch et := e.(type) {
			case *watcherx.ErrorEvent:
				p.runOnChanges(e, et)
				continue
			default:
				nk, _, cancel, err := p.forkKoanf()
				if err != nil {
					p.runOnChanges(e, err)
					continue
				}

				var cancelReload bool
				for _, key := range p.immutables {
					if !reflect.DeepEqual(k.Get(key), nk.Get(key)) {
						cancel()
						cancelReload = true
						p.runOnChanges(e, NewImmutableError(key, fmt.Sprintf("%v", k.Get(key)), fmt.Sprintf("%v", nk.Get(key))))
						break
					}
				}

				if cancelReload {
					continue
				}

				p.replaceKoanf(nk, cancel)
				p.runOnChanges(e, nil)
			}
		}
	}

	for _, path := range paths {
		fp, err := NewKoanfFile(ctx, path)
		if err != nil {
			return err
		}

		if err := k.Load(fp, nil); err != nil {
			return err
		}

		c := make(watcherx.EventChannel)
		if _, err := fp.WatchChannel(c); err != nil {
			return err
		}

		go watchForFileChanges(c)
	}

	return nil
}

func (p *Provider) Set(key string, value interface{}) error {
	p.forcedValues = append(p.forcedValues, tuple{Key: key, Value: value})

	k, _, cancel, err := p.forkKoanf()
	if err != nil {
		return err
	}

	p.replaceKoanf(k, cancel)
	return nil
}

func (p *Provider) BoolF(key string, fallback bool) bool {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Bool(key)
}

func (p *Provider) StringF(key string, fallback string) string {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.String(key)
}

func (p *Provider) StringsF(key string, fallback []string) (val []string) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Strings(key)
}

func (p *Provider) IntF(key string, fallback int) (val int) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Int(key)
}

func (p *Provider) Float64F(key string, fallback float64) (val float64) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Float64(key)
}

func (p *Provider) DurationF(key string, fallback time.Duration) (val time.Duration) {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	return p.Duration(key)
}

func (p *Provider) ByteSizeF(key string, fallback bytesize.ByteSize) bytesize.ByteSize {
	if !p.Koanf.Exists(key) {
		return fallback
	}

	switch v := p.Koanf.Get(key).(type) {
	case string:
		// this type usually comes from user input
		dec, err := bytesize.Parse(v)
		if err != nil {
			p.logger.WithField("key", key).WithField("raw_value", v).WithError(err).Warnf("error parsing byte size value, using fallback of %s", fallback)
			return fallback
		}
		return dec
	case float64:
		// this type comes from json.Unmarshal
		return bytesize.ByteSize(v)
	case bytesize.ByteSize:
		return v
	default:
		p.logger.WithField("key", key).WithField("raw_type", fmt.Sprintf("%T", v)).WithField("raw_value", fmt.Sprintf("%+v", v)).Errorf("error converting byte size value because of unknown type, using fallback of %s", fallback)
		return fallback
	}
}

func (p *Provider) GetF(key string, fallback interface{}) (val interface{}) {
	if !p.Exists(key) {
		return fallback
	}

	return p.Get(key)
}

func (p *Provider) CORS(prefix string, defaults cors.Options) (cors.Options, bool) {
	if len(prefix) > 0 {
		prefix = strings.TrimRight(prefix, ".") + "."
	}

	return cors.Options{
		AllowedOrigins:     p.StringsF(prefix+"cors.allowed_origins", defaults.AllowedOrigins),
		AllowedMethods:     p.StringsF(prefix+"cors.allowed_methods", defaults.AllowedMethods),
		AllowedHeaders:     p.StringsF(prefix+"cors.allowed_headers", defaults.AllowedHeaders),
		ExposedHeaders:     p.StringsF(prefix+"cors.exposed_headers", defaults.ExposedHeaders),
		AllowCredentials:   p.BoolF(prefix+"cors.allow_credentials", defaults.AllowCredentials),
		OptionsPassthrough: p.BoolF(prefix+"cors.options_passthrough", defaults.OptionsPassthrough),
		MaxAge:             p.IntF(prefix+"cors.max_age", defaults.MaxAge),
		Debug:              p.BoolF(prefix+"cors.debug", defaults.Debug),
	}, p.Bool(prefix + "cors.enabled")
}

func (p *Provider) TracingConfig(serviceName string) *tracing.Config {
	return &tracing.Config{
		ServiceName: p.StringF("tracing.service_name", serviceName),
		Provider:    p.String("tracing.provider"),
		Jaeger: &tracing.JaegerConfig{
			LocalAgentHostPort: p.String("tracing.providers.jaeger.local_agent_address"),
			SamplerType:        p.StringF("tracing.providers.jaeger.sampling.type", "const"),
			SamplerValue:       p.Float64F("tracing.providers.jaeger.sampling.value", float64(1)),
			SamplerServerURL:   p.String("tracing.providers.jaeger.sampling.server_url"),
			Propagation: stringsx.Coalesce(
				os.Getenv("JAEGER_PROPAGATION"),
				p.String("tracing.providers.jaeger.propagation"),
			),
		},
		Zipkin: &tracing.ZipkinConfig{
			ServerURL: p.String("tracing.providers.zipkin.server_url"),
		},
	}
}

func (p *Provider) RequestURIF(path string, fallback *url.URL) *url.URL {
	switch t := p.Get(path).(type) {
	case *url.URL:
		return t
	case url.URL:
		return &t
	case string:
		if parsed, err := url.ParseRequestURI(t); err == nil {
			return parsed
		}
	}

	return fallback
}

func (p *Provider) URIF(path string, fallback *url.URL) *url.URL {
	switch t := p.Get(path).(type) {
	case *url.URL:
		return t
	case url.URL:
		return &t
	case string:
		if parsed, err := url.Parse(t); err == nil {
			return parsed
		}
	}

	return fallback
}

// PrintHumanReadableValidationErrors prints human readable validation errors. Duh.
func (p *Provider) PrintHumanReadableValidationErrors(w io.Writer, err error) {
	p.printHumanReadableValidationErrors(p.Koanf, w, err)
}

func (p *Provider) printHumanReadableValidationErrors(k *koanf.Koanf, w io.Writer, err error) {
	if err == nil {
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, "")
	conf, innerErr := k.Marshal(json.Parser())
	if innerErr != nil {
		_, _ = fmt.Fprintf(w, "Unable to unmarshal configuration: %+v", innerErr)
	}

	jsonschemax.FormatValidationErrorForCLI(w, conf, err)
}

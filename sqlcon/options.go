package sqlcon

type options struct {
	UseTracedDriver  bool
	OmitArgs         bool
	AllowRoot        bool
	forcedDriverName string
}

type Opt func(*options)

// WithDistributedTracing will make it so that a wrapped driver is used that supports the opentracing API
func WithDistributedTracing() Opt {
	return func(o *options) {
		o.UseTracedDriver = true
	}
}

// WithOmitArgsFromTraceSpans will make it so that query arguments are omitted from tracing spans
func WithOmitArgsFromTraceSpans() Opt {
	return func(o *options) {
		o.OmitArgs = true
	}
}

// WithAllowRoot will make it so that root spans will be created if a trace could not be found using
// opentracing's SpanFromContext method
func WithAllowRoot() Opt {
	return func(o *options) {
		o.AllowRoot = true
	}
}

// This option is specifically for writing tests as you can't register a driver with the same name more than once
func WithRandomDriverName() Opt {
	return func(o *options) {
		o.forcedDriverName = uuid.NewV4().String()
	}
}

package sqlcon

type options struct {
	UseTracedDriver bool
	OmitArgs        bool
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

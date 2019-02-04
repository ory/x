package dbal

type DriverOptionModifier func(*DriverOption)

type DriverOption struct {
	UseTracing bool

	// unexported as these are specifically used by Hydra for writing tests
	useRandomDriverName   bool
	allowRootTracingSpans bool
	omitSQLArgsFromSpans  bool
}

// WithTracing will make it so that a wrapped driver is used that supports the OpenTracing API
func WithTracing() DriverOptionModifier {
	return func(o *DriverOption) {
		o.UseTracing = true
	}
}

// this option is specifically for writing tests as you can't register a driver with the same name more than once
func withRandomDriverName() DriverOptionModifier {
	return func(o *DriverOption) {
		o.useRandomDriverName = true
	}
}

// withAllowRoot will make it so that root spans will be created if a trace could not be found in the context
func withAllowRootTraceSpans() DriverOptionModifier {
	return func(o *DriverOption) {
		o.allowRootTracingSpans = true
	}
}

// withOmitSQLArgsFromSpans will make it so that query arguments are omitted from tracing spans
func withOmitSQLArgsFromSpans() DriverOptionModifier {
	return func(o *DriverOption) {
		o.omitSQLArgsFromSpans = true
	}
}


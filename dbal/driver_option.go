package dbal

// DriverOptionModifier is a function modifying a DriverOption.
type DriverOptionModifier func(*DriverOption)

// DriverOption encapsulates DBAL driver options.
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

// WithRandomDriverName is specifically for writing tests as you can't register a driver with the same name more than once
func WithRandomDriverName() DriverOptionModifier {
	return func(o *DriverOption) {
		o.useRandomDriverName = true
	}
}

// WithAllowRootTraceSpans will make it so that root spans will be created if a trace could not be found in the context
func WithAllowRootTraceSpans() DriverOptionModifier {
	return func(o *DriverOption) {
		o.allowRootTracingSpans = true
	}
}

// WithOmitSQLArgsFromSpans will make it so that query arguments are omitted from tracing spans
func WithOmitSQLArgsFromSpans() DriverOptionModifier {
	return func(o *DriverOption) {
		o.omitSQLArgsFromSpans = true
	}
}

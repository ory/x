package tracing

import datadogTracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

type datadogCloser struct {
}

func (t datadogCloser) Close() error {
	datadogTracer.Stop()
	return nil
}

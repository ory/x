// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package tracing

import datadogTracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

type datadogCloser struct {
}

func (t datadogCloser) Close() error {
	datadogTracer.Stop()
	return nil
}

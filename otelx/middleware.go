package otelx

import (
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewHandler(handler http.Handler, operation string) http.Handler {
	healthFilter := func(r *http.Request) bool {
		path := r.URL.Path
		if strings.HasPrefix(path, "/health/") {
			return false
		}
		return true
	}

	return otelhttp.NewHandler(handler, operation, otelhttp.WithFilter(
		healthFilter,
	))
}

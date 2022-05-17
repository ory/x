package otelx

import (
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const tracingComponent = "github.com/ory/x/otelx"

func isHealthFilter(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/health/") {
		return false
	}
	return true
}

func isAdminHealthFilter(r *http.Request) bool {
	path := r.URL.Path
	if strings.HasPrefix(path, "/admin/health/") {
		return false
	}
	return true
}

func filterOpts() []otelhttp.Option {
	filters := []otelhttp.Filter{
		isHealthFilter,
		isAdminHealthFilter,
	}
	opts := []otelhttp.Option{}
	for _, f := range filters {
		opts = append(opts, otelhttp.WithFilter(f))
	}
	return opts
}

func NewHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation, filterOpts()...)
}

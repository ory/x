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

// NewHandler returns a wrapped otelhttp.NewHandler with our request filters.
func NewHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation, filterOpts()...)
}

// TraceHandler wraps otelx.NewHandler, passing the URL path as the span name.
func TraceHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		NewHandler(h, r.URL.Path).ServeHTTP(w, r)
	})
}

package otelx

import (
	"context"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
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

// Middleware to satisfy httprouter.Handle. Pass a http.Handler from NewHandler
// here.
func WrapHTTPRouter(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		newCtx := context.WithValue(ctx, "params", ps)
		r = r.WithContext(newCtx)
		h.ServeHTTP(w, r)
	}
}

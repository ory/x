package prometheus

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ory/herodot"
)

const (
	MetricsPrometheusPath = "/metrics/prometheus"
)

// Handler handles HTTP requests to health and version endpoints.
type Handler struct {
	H             herodot.Writer
	VersionString string
}

// NewHandler instantiates a handler.
func NewHandler(
	h herodot.Writer,
	version string,
) *Handler {
	return &Handler{
		H:             h,
		VersionString: version,
	}
}

// SetRoutes registers this handler's routes.
func (h *Handler) SetRoutes(r *httprouter.Router) {
	r.GET(MetricsPrometheusPath, h.Metrics)
}

// Metrics outputs prometheus metrics
//
// swagger:route GET /metrics/prometheus admin prometheus
//
// Get snapshot metrics from the Hydra service. If you're using k8s, you can then add annotations to
// your deployment like so:
//
// ```
// metadata:
//  annotations:
//    prometheus.io/port: "4434"
//      prometheus.io/path: "/metrics/prometheus"
// ```
//
//     Produces:
//     - plain/text
//
//     Responses:
//       200: emptyResponse
func (h *Handler) Metrics(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	promhttp.Handler().ServeHTTP(rw, r)
}

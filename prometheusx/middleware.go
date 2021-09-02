package prometheus

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type MetricsManager struct {
	prometheusMetrics *Metrics
	routers           []*httprouter.Router
}

func NewMetricsManager(app, version, hash, buildTime string) *MetricsManager {
	return NewMetricsManagerWithPrefix(app, "", version, hash, buildTime)
}

// NewMetricsManagerWithPrefix creates MetricsManager that uses metricsPrefix parameters as a prefix
// for all metrics registered within this middleware. Constants HttpMetrics or GrpcMetrics can be used
// respectively. Setting empty string in metricsPrefix will be equivalent to calling NewMetricsManager.
func NewMetricsManagerWithPrefix(app, metricsPrefix, version, hash, buildTime string) *MetricsManager {
	return &MetricsManager{
		prometheusMetrics: NewMetrics(app, metricsPrefix, version, hash, buildTime),
	}
}

// Main middleware method to collect metrics for Prometheus.
func (pmm *MetricsManager) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	pmm.prometheusMetrics.Instrument(rw, next, pmm.getLabelForPath(r))(rw, r)
}

func (pmm *MetricsManager) RegisterRouter(router *httprouter.Router) {
	pmm.routers = append(pmm.routers, router)
}

func (pmm *MetricsManager) getLabelForPath(r *http.Request) string {
	// looking for a match in one of registered routers
	for _, router := range pmm.routers {
		handler, params, _ := router.Lookup(r.Method, r.URL.Path)
		if handler != nil {
			return reconstructEndpoint(r.URL.Path, params)
		}
	}
	return "{unmatched}"
}

// To reduce cardinality of labels, values of matched path parameters must be replaced with {param}
func reconstructEndpoint(path string, params httprouter.Params) string {
	// if map is empty, then nothing to change in the path
	if len(params) == 0 {
		return path
	}

	// construct a list of parameter values
	paramValues := make(map[string]struct{}, len(params))
	for _, param := range params {
		paramValues[param.Value] = struct{}{}
	}

	parts := strings.Split(path, "/")
	for index, part := range parts {
		if _, ok := paramValues[part]; ok {
			parts[index] = "{param}"
		}
	}

	return strings.Join(parts, "/")
}

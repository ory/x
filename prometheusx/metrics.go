package prometheus

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics prototypes
type Metrics struct {
	ResponseTime *prometheus.HistogramVec
}

// Method for creation new custom Prometheus  metrics
func NewMetrics(app, version, hash, date string) *Metrics {
	pm := &Metrics{
		ResponseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: app + "_response_time_seconds",
				Help: "Description",
				ConstLabels: map[string]string{
					"version":   version,
					"hash":      hash,
					"buildTime": date,
				},
			},
			[]string{"endpoint"},
		),
	}
	err := prometheus.Register(pm.ResponseTime)
	if e := new(prometheus.AlreadyRegisteredError); errors.As(err, e) {
		return pm
	} else if err != nil {
		panic(err)
	}

	return pm
}

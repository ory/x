package metricsx

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type SegmentCounter struct {
	m prometheus.Counter
}

func (c *SegmentCounter) Inc() {
	c.m.Inc()
}

func (c *SegmentCounter) Add(val float64) {
	log.Println("SegmentCounter.Add()")
	c.m.Add(val)
}

func (c *SegmentCounter) Desc() *prometheus.Desc {
	return c.m.Desc()
}

func (c *SegmentCounter) Write(m *dto.Metric) error {
	return c.m.Write(m)
}

func (c *SegmentCounter) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentCounter) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func WithSegmentCounter(m prometheus.Counter) *SegmentCounter {
	return &SegmentCounter{
		m: m,
	}
}

type SegmentCounterVec struct {
	m *prometheus.CounterVec
}

func (c *SegmentCounterVec) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentCounterVec) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func (c *SegmentCounterVec) With(labels prometheus.Labels) prometheus.Counter {
	return WithSegmentCounter(c.m.With(labels))
}

func (c *SegmentCounterVec) WithLabelValues(lvs ...string) prometheus.Counter {
	return WithSegmentCounter(c.m.WithLabelValues(lvs...))
}

func WithSegmentCounterVec(m *prometheus.CounterVec) *SegmentCounterVec {
	return &SegmentCounterVec{
		m: m,
	}
}

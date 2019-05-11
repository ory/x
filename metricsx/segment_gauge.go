package metricsx

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type SegmentGauge struct {
	m prometheus.Gauge
}

func (c *SegmentGauge) Inc() {
	log.Println("SegmentGauge.Inc()")
	c.m.Inc()
}

func (c *SegmentGauge) Dec() {
	c.m.Dec()
}

func (c *SegmentGauge) Set(val float64) {
	log.Println("SegmentGauge.Set()")
	c.m.Set(val)
}

func (c *SegmentGauge) Add(val float64) {
	log.Println("SegmentGauge.Add()")
	c.m.Add(val)
}

func (c *SegmentGauge) Sub(val float64) {
	log.Println("SegmentGauge.Sub()")
	c.m.Sub(val)
}

func (c *SegmentGauge) SetToCurrentTime() {
	c.m.SetToCurrentTime()
}

func (c *SegmentGauge) Desc() *prometheus.Desc {
	return c.m.Desc()
}

func (c *SegmentGauge) Write(m *dto.Metric) error {
	return c.m.Write(m)
}

func (c *SegmentGauge) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentGauge) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func WithSegmentGauge(m prometheus.Gauge) *SegmentGauge {
	return &SegmentGauge{
		m: m,
	}
}

type SegmentGaugeVec struct {
	m *prometheus.GaugeVec
}

func (c *SegmentGaugeVec) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentGaugeVec) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func (c *SegmentGaugeVec) With(labels prometheus.Labels) prometheus.Gauge {
	return WithSegmentGauge(c.m.With(labels))
}

func (c *SegmentGaugeVec) WithLabelValues(lvs ...string) prometheus.Gauge {
	return WithSegmentGauge(c.m.WithLabelValues(lvs...))
}

func WithSegmentGaugeVec(m *prometheus.GaugeVec) *SegmentGaugeVec {
	return &SegmentGaugeVec{
		m: m,
	}
}

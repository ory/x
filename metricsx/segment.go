package metricsx

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/segmentio/analytics-go"
)

// A SegmentCounter is a prometheus Counter that sends data to Segment
type SegmentCounter struct {
	m prometheus.Counter
	o prometheus.CounterOpts
	l analytics.Properties
}

// Inc sends a single metric to segment and increments the metric value by 1.
func (c *SegmentCounter) Inc() {
	if segmentInstance != nil {
		go segmentCounterInc(c)
	}

	c.m.Inc()
}

// Add adds `val` to the total value of the metric, but calls sends n different "increment" metrics to segment, where n=val
func (c *SegmentCounter) Add(val float64) {
	if segmentInstance != nil {
		for i := 0; i < int(val); i++ {
			go segmentCounterInc(c)
		}
	}
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

// A SegmentCounterVec is a prometheus Counter with labels that sends data to Segment
type SegmentCounterVec struct {
	m CounterVec
	o prometheus.CounterOpts
}

func (c *SegmentCounterVec) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentCounterVec) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func (c *SegmentCounterVec) With(labels prometheus.Labels) prometheus.Counter {
	return WithSegmentCounter(c.o, labels, c.m.With(labels))
}

// A SegmentGaugeVec is a prometheus Gauge that sends data to Segment
type SegmentGauge struct {
	m prometheus.Gauge
	o prometheus.GaugeOpts
	l analytics.Properties
}

func (c *SegmentGauge) Inc() {
	if segmentInstance != nil {
		go func(c *SegmentGauge) {
			v := &dto.Metric{}
			if err := c.Write(v); err != nil {
				segmentInstance.l.WithError(err).Debug("Could not commit read prometheus metric")
			} else {
				value := v.GetGauge().GetValue() + 1
				segmentGaugeSet(c, value)
			}
		}(c)
	}
	c.m.Inc()
}

func (c *SegmentGauge) Dec() {
	if segmentInstance != nil {
		go func(c *SegmentGauge) {
			v := &dto.Metric{}
			if err := c.Write(v); err != nil {
				segmentInstance.l.WithError(err).Debug("Could not commit read prometheus metric")
			} else {
				value := v.GetGauge().GetValue() - 1
				segmentGaugeSet(c, value)
			}
		}(c)
	}
	c.m.Dec()
}

func (c *SegmentGauge) Set(val float64) {
	if segmentInstance != nil {
		go segmentGaugeSet(c, val)
	}
	c.m.Set(val)
}

func (c *SegmentGauge) Add(val float64) {
	if segmentInstance != nil {
		go func(c *SegmentGauge) {
			v := &dto.Metric{}
			if err := c.Write(v); err != nil {
				segmentInstance.l.WithError(err).Debug("Could not commit read prometheus metric")
			} else {
				value := v.GetGauge().GetValue() + val
				segmentGaugeSet(c, value)
			}
		}(c)
	}
	c.m.Add(val)
}

func (c *SegmentGauge) Sub(val float64) {
	if segmentInstance != nil {
		go func(c *SegmentGauge) {
			v := &dto.Metric{}
			if err := c.Write(v); err != nil {
				segmentInstance.l.WithError(err).Debug("Could not commit read prometheus metric")
			} else {
				value := v.GetGauge().GetValue() - val
				segmentGaugeSet(c, value)
			}
		}(c)
	}
	c.m.Sub(val)
}

func (c *SegmentGauge) SetToCurrentTime() {
	if segmentInstance != nil {
		go segmentGaugeSet(c, float64(time.Now().Unix()))
	}
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

// A SegmentGaugeVec is a prometheus Gauge with labels that sends data to Segment
type SegmentGaugeVec struct {
	m GaugeVec
	o prometheus.GaugeOpts
}

func (c *SegmentGaugeVec) Describe(m chan<- *prometheus.Desc) {
	c.m.Describe(m)
}

func (c *SegmentGaugeVec) Collect(m chan<- prometheus.Metric) {
	c.m.Collect(m)
}

func (c *SegmentGaugeVec) With(labels prometheus.Labels) prometheus.Gauge {
	return WithSegmentGauge(c.o, labels, c.m.With(labels))
}

func WithSegmentCounter(o prometheus.CounterOpts, l prometheus.Labels, m prometheus.Counter) *SegmentCounter {
	return &SegmentCounter{
		m: m,
		o: o,
		l: propertiesFromLabels(l),
	}
}

func WithSegmentCounterVec(o prometheus.CounterOpts, m CounterVec) *SegmentCounterVec {
	return &SegmentCounterVec{
		m: m,
		o: o,
	}
}

func WithSegmentGauge(o prometheus.GaugeOpts, l prometheus.Labels, m prometheus.Gauge) *SegmentGauge {
	return &SegmentGauge{
		m: m,
		o: o,
		l: propertiesFromLabels(l),
	}
}

func WithSegmentGaugeVec(o prometheus.GaugeOpts, m GaugeVec) *SegmentGaugeVec {
	return &SegmentGaugeVec{
		m: m,
		o: o,
	}
}

func getMetricName(namespace string, subsystem string, name string) string {
	s := make([]string, 3)
	i := 0
	if namespace != "" {
		s[i] = namespace
		i++
	}

	if subsystem != "" {
		s[i] = subsystem
		i++
	}

	s[i] = name

	return strings.Join(s, "_")
}

func getGaugeName(o prometheus.GaugeOpts) string {
	return getMetricName(o.Namespace, o.Subsystem, o.Name)
}

func getCounterName(o prometheus.CounterOpts) string {
	return getMetricName(o.Namespace, o.Subsystem, o.Name)
}

func propertiesFromLabels(l prometheus.Labels) analytics.Properties {
	p := analytics.Properties{}
	for k, v := range l {
		p[k] = v
	}
	return p
}

func segmentCounterInc(c *SegmentCounter) {
	if err := segmentInstance.c.Enqueue(analytics.Track{
		UserId:     segmentInstance.o.ClusterID,
		Event:      getCounterName(c.o),
		Context:    segmentInstance.context,
		Properties: c.l,
	}); err != nil {
		segmentInstance.l.WithError(err).Debug("Could not commit anonymized telemetry data")
	}

	segmentInstance.l.WithField("metric", c.o.Name).Infoln("Finished incrementing metric for segment")
}

func segmentGaugeSet(c *SegmentGauge, value float64) {
	c.l.Set("value", value)
	if err := segmentInstance.c.Enqueue(analytics.Track{
		UserId:     segmentInstance.o.ClusterID,
		Event:      getGaugeName(c.o),
		Context:    segmentInstance.context,
		Properties: c.l,
	}); err != nil {
		segmentInstance.l.WithError(err).Debug("Could not commit anonymized telemetry data")
	}

	segmentInstance.l.WithField("metric", c.o.Name).Infoln("Finished setting metric for segment")
}

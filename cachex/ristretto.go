package cachex

import (
	"github.com/dgraph-io/ristretto/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// RistrettoCollector collects Ristretto cache metrics.
type RistrettoCollector[K ristretto.Key, V any] struct {
	cache   *ristretto.Cache[K, V]
	prefix  string
	suffix  string
	metrics *ristretto.Metrics
}

// NewRistrettoCollector creates a new RistrettoCollector.
//
// To use this collector, you need to register it with a Prometheus registry:
//
//	func main() {
//		cache, _ := ristretto.NewCache(&ristretto.Config{
//			NumCounters: 1e7,
//			MaxCost:     1 << 30,
//			BufferItems: 64,
//		})
//		collector := NewRistrettoCollector(cache, "prefix_", "_suffix")
//		prometheus.MustRegister(collector)
//	}
func NewRistrettoCollector[K ristretto.Key, V any](cache *ristretto.Cache[K, V], prefix string, suffix string) *RistrettoCollector[K, V] {
	return &RistrettoCollector[K, V]{
		cache:   cache,
		prefix:  prefix,
		suffix:  suffix,
		metrics: cache.Metrics,
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector.
func (c *RistrettoCollector[K, V]) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(c.prefix+"ristretto_hits"+c.suffix, "Total number of cache hits", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_misses"+c.suffix, "Total number of cache misses", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_ratio"+c.suffix, "Cache hit ratio", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_keys_added"+c.suffix, "Total number of keys added to the cache", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_cost_added"+c.suffix, "Total cost of keys added to the cache", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_keys_evicted"+c.suffix, "Total number of keys evicted from the cache", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_cost_evicted"+c.suffix, "Total cost of keys evicted from the cache", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_sets_dropped"+c.suffix, "Total number of sets dropped", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_sets_rejected"+c.suffix, "Total number of sets rejected", nil, nil)
	ch <- prometheus.NewDesc(c.prefix+"ristretto_gets_kept"+c.suffix, "Total number of gets kept", nil, nil)
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *RistrettoCollector[K, V]) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_hits"+c.suffix, "Total number of cache hits", nil, nil), prometheus.GaugeValue, float64(c.metrics.Hits()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_misses"+c.suffix, "Total number of cache misses", nil, nil), prometheus.GaugeValue, float64(c.metrics.Misses()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_ratio"+c.suffix, "Cache hit ratio", nil, nil), prometheus.GaugeValue, c.metrics.Ratio())
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_keys_added"+c.suffix, "Total number of keys added to the cache", nil, nil), prometheus.GaugeValue, float64(c.metrics.KeysAdded()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_cost_added"+c.suffix, "Total cost of keys added to the cache", nil, nil), prometheus.GaugeValue, float64(c.metrics.CostAdded()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_keys_evicted"+c.suffix, "Total number of keys evicted from the cache", nil, nil), prometheus.GaugeValue, float64(c.metrics.KeysEvicted()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_cost_evicted"+c.suffix, "Total cost of keys evicted from the cache", nil, nil), prometheus.GaugeValue, float64(c.metrics.CostEvicted()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_sets_dropped"+c.suffix, "Total number of sets dropped", nil, nil), prometheus.GaugeValue, float64(c.metrics.SetsDropped()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_sets_rejected"+c.suffix, "Total number of sets rejected", nil, nil), prometheus.GaugeValue, float64(c.metrics.SetsRejected()))
	ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(c.prefix+"ristretto_gets_kept"+c.suffix, "Total number of gets kept", nil, nil), prometheus.GaugeValue, float64(c.metrics.GetsKept()))
}

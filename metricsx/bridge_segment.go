package metricsx

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"

	"github.com/ory/x/resilience"
)

type analyticsProperties struct {
	memstats analytics.Properties
	app      analytics.Properties
}

// the []string key in this map is the list of
var memStats = map[string]string{
	"go_memstats_heap_alloc_bytes":    "alloc",
	"go_memstats_alloc_bytes_total":   "totalAlloc",
	"go_memstats_sys_bytes":           "sys",
	"go_memstats_lookups_total":       "lookups",
	"go_memstats_mallocs_total":       "mallocs",
	"go_memstats_frees_total":         "frees",
	"go_memstats_heap_sys_bytes":      "heapSys",
	"go_memstats_heap_idle_bytes":     "heapIdle",
	"go_memstats_heap_inuse_bytes":    "heapInuse",
	"go_memstats_heap_released_bytes": "heapReleased",
	"go_memstats_heap_objects":        "heapObjects",
	"go_gc_duration_seconds_count":    "numGC",
}

func labelsContain(key string, labels []*dto.LabelPair) bool {
	for _, v := range labels {
		if v.GetName() == key {
			return true
		}
	}
	return false
}

func getSegmentKey(mf *dto.MetricFamily) (string, bool) {
	if val, ok := memStats[mf.GetName()]; ok {
		return val, true
	}
	return mf.GetName(), false
}

func getValueFromMetric(t dto.MetricType, m *dto.Metric) float64 {
	switch t {
	case dto.MetricType_GAUGE:
		return m.GetGauge().GetValue()
	case dto.MetricType_COUNTER:
		return m.GetCounter().GetValue()
	}
	return 0
}

func getSegmentProperties(mfs []*dto.MetricFamily) analyticsProperties {
	p := analyticsProperties{
		memstats: analytics.NewProperties(),
		app:      analytics.NewProperties(),
	}
	for _, v := range mfs {
		for _, m := range v.GetMetric() {
			if v.GetType() == dto.MetricType_HISTOGRAM || v.GetType() == dto.MetricType_SUMMARY {
				// Don't transfer histogram buckets / summary percentiles to segment
				if labelsContain("percentile", m.GetLabel()) || labelsContain("le", m.GetLabel()) {
					continue
				}
			}

			value := getValueFromMetric(v.GetType(), m)
			if key, ok := getSegmentKey(v); ok {
				p.memstats.Set(key, value)
			} else {
				p.app.Set(v.GetName(), value)
			}
		}
	}

	return p
}

// SegmentOptions provides configuration settings for the FormattedSegmentBridge
type SegmentOptions struct {
	// Service represents the service name, for example "ory-hydra".
	Service string

	// ClusterID represents the cluster id, typically a hash of some unique configuration properties.
	ClusterID string

	// IsDevelopment should be true if we assume that we're in a development environment.
	IsDevelopment bool

	// WriteKey is the segment API key.
	WriteKey string

	// WhitelistedPaths represents a list of paths that can be transmitted in clear text to segment.
	WhitelistedPaths []string

	// BuildVersion represents the build version.
	BuildVersion string

	// BuildHash represents the build git hash.
	BuildHash string

	// BuildTime represents the build time.
	BuildTime string
}

type FormattedSegmentBridge struct {
	o       *SegmentOptions
	client  analytics.Client
	context *analytics.Context
	g       prometheus.Gatherer
	l       logrus.FieldLogger

	salt string
}

func (s *FormattedSegmentBridge) Push(ctx context.Context) error {
	mfs, err := s.g.Gather()
	if err != nil {
		return err
	}

	p := getSegmentProperties(mfs)

	if err := s.client.Enqueue(analytics.Track{
		UserId:     s.o.ClusterID,
		Event:      "memstats",
		Properties: p.memstats,
		Context:    s.context,
	}); err != nil {
		s.l.WithError(err).Debug("Unable to enqueue memstats metrics")
	}

	if err := s.client.Enqueue(analytics.Track{
		UserId:     s.o.ClusterID,
		Event:      s.o.Service,
		Properties: p.app,
		Context:    s.context,
	}); err != nil {
		s.l.WithError(err).Debug("Unable to enqueue app metrics")
	}

	return nil
}

func (s *FormattedSegmentBridge) enqueueMetric(name string, t dto.MetricType, m *dto.Metric) error {
	p := analytics.Properties{}
	val := getValueFromMetric(t, m)
	p.Set("_value", val)
	log.Println("_value=", val)
	err := s.client.Enqueue(analytics.Track{
		UserId:     s.o.ClusterID,
		Properties: p,
		Context:    s.context,
		Event:      name,
	})

	if err != nil {
		log.Println(err)
	}

	return nil
}

// ServeHTTP is a middleware for sending meta information to segment.
func (s *FormattedSegmentBridge) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	next(rw, r)

	latency := time.Now().UTC().Sub(start.UTC()) / time.Millisecond

	enqueue(rw, r, s.client, s.l, s.o.WhitelistedPaths, latency, s.salt, s.o.ClusterID, s.context)
}

func (s *FormattedSegmentBridge) anonymizePath(path string, salt string) string {
	paths := s.o.WhitelistedPaths
	path = strings.ToLower(path)

	for _, p := range paths {
		p = strings.ToLower(p)
		if len(path) == len(p) && path[:len(p)] == strings.ToLower(p) {
			return p
		} else if len(path) > len(p) && path[:len(p)+1] == strings.ToLower(p)+"/" {
			return path[:len(p)] + "/" + Hash(path[len(p):]+"|"+salt)
		}
	}

	return "/"
}

func (s *FormattedSegmentBridge) anonymizeQuery(query url.Values, salt string) string {
	for _, q := range query {
		for i, s := range q {
			if s != "" {
				s = Hash(s + "|" + salt)
				q[i] = s
			}
		}
	}
	return query.Encode()
}

func NewFormattedSegmentBridge(ctx context.Context, o *SegmentOptions, logger logrus.FieldLogger, gatherer prometheus.Gatherer) (*FormattedSegmentBridge, error) {
	client, err := analytics.NewWithConfig(o.WriteKey, analytics.Config{
		Interval:  time.Hour * 24,
		BatchSize: 100,
	})

	if err != nil {
		return nil, err
	}

	oi := analytics.OSInfo{
		Version: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
	}
	actx := &analytics.Context{
		IP: net.IPv4(0, 0, 0, 0),
		App: analytics.AppInfo{
			Name:    o.Service,
			Version: o.BuildVersion,
			Build:   fmt.Sprintf("%s/%s/%s", o.BuildVersion, o.BuildHash, o.BuildTime),
		},
		OS: oi,
		Traits: analytics.NewTraits().
			Set("optedOut", false).
			Set("instanceId", uuid.New()).
			Set("isDevelopment", o.IsDevelopment),
		UserAgent: "github.com/ory/x/metricsx.Service/v0.0.1",
	}

	if err := resilience.Retry(logger, time.Minute*5, time.Hour*24*30, func() error {
		return client.Enqueue(analytics.Identify{
			UserId:  o.ClusterID,
			Traits:  actx.Traits,
			Context: actx,
		})
	}); err != nil {
		logger.WithError(err).Debug("Could not commit anonymized environment information")
	}
	return &FormattedSegmentBridge{
		client: client,
		g:      gatherer,
		o:      o,
		l:      logger,
		salt:   uuid.New(),
	}, nil
}

package metricsx

import (
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ory/x/cmdx"
	"github.com/pborman/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
)

var DefaultWatchDuration = time.Hour * 12

type void struct {
}

func (v *void) Logf(format string, args ...interface{}) {
}

func (v *void) Errorf(format string, args ...interface{}) {
}

// Options configures the metrics service.
type Options struct {
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

	// Config overrides the analytics.Config. If nil, sensible defaults will be used.
	Config *analytics.Config

	// MemoryInterval sets how often memory statistics should be transmitted. Defaults to every 12 hours.
	MemoryInterval time.Duration
}

// The Observer type is used for populating data into prometheus metrics that are repeatedly pulled from the database.
type Observer interface {
	Observe() error
}

// Watch will continuously call o.Observe every time `i` ticks.
func Watch(o Observer, i time.Duration, log logrus.FieldLogger) {
	t := time.NewTicker(i)

	for {
		select {
		case <-t.C:
			if err := o.Observe(); err != nil {
				log.WithError(err).Infoln("Error when gathering metrics")
			}
		}
	}
}

// Service helps with providing context on metrics.
type Service struct {
	optOut bool
	salt   string

	o       *Options
	context *analytics.Context

	c analytics.Client
	l logrus.FieldLogger

	mem *MemoryStatistics
}

// New returns a new metrics service. If one has been instantiated already, no new instance will be created.
// A lock is used because the value of the segmentInstance (a shared pointer) is being mutated.
func New(
	cmd *cobra.Command,
	l logrus.FieldLogger,
	o *Options,
) *Service {
	lock.Lock()
	defer lock.Unlock()

	if segmentInstance != nil {
		return segmentInstance
	}

	if o.BuildTime == "" {
		o.BuildTime = "unknown"
	}

	if o.BuildVersion == "" {
		o.BuildVersion = "unknown"
	}

	if o.BuildHash == "" {
		o.BuildHash = "unknown"
	}

	if o.Config == nil {
		o.Config = &analytics.Config{
			Interval:  time.Hour * 24,
			BatchSize: 100,
		}
	}

	o.Config.Logger = new(void)

	if o.MemoryInterval < time.Minute {
		o.MemoryInterval = DefaultWatchDuration
	}

	segment, err := analytics.NewWithConfig(o.WriteKey, *o.Config)
	if err != nil {
		l.WithError(err).Fatalf("Unable to initialise software quality assurance features.")
		return nil
	}

	var oi analytics.OSInfo

	optOut := isOptedOut(cmd, l)

	if !optOut {
		l.Info("Software quality assurance features are enabled. Learn more at: https://www.ory.sh/docs/ecosystem/sqa")
		oi = analytics.OSInfo{
			Version: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
		}
	}

	m := &Service{
		optOut: optOut,
		salt:   uuid.New(),
		o:      o,
		c:      segment,
		l:      l,
		mem:    new(MemoryStatistics),
		context: &analytics.Context{
			IP: net.IPv4(0, 0, 0, 0),
			App: analytics.AppInfo{
				Name:    o.Service,
				Version: o.BuildVersion,
				Build:   fmt.Sprintf("%s/%s/%s", o.BuildVersion, o.BuildHash, o.BuildTime),
			},
			OS: oi,
			Traits: analytics.NewTraits().
				Set("optedOut", optOut).
				Set("instanceId", uuid.New()).
				Set("isDevelopment", o.IsDevelopment),
			UserAgent: "github.com/ory/x/metricsx.Service/v0.0.1",
		},
	}

	segmentInstance = m

	go m.Identify()
	go m.ObserveMemory()

	return m
}

// GaugeVec is an interface which prometheus.GaugeVec implements.
// This allows for more generic wrapping of prometheus.GaugeVec
type GaugeVec interface {
	prometheus.Collector
	With(prometheus.Labels) prometheus.Gauge
}

// CounterVec is an interface which prometheus.CounterVec implements.
// This allows for more generic wrapping of prometheus.CounterVec
type CounterVec interface {
	prometheus.Collector
	With(prometheus.Labels) prometheus.Counter
}

// NewGaugeVec returns a GaugeVec. This function uses options provided to the process to create
// the appropriate wrappers.
// As of writing this, only Segment is available. However, in the future, it is likely that many different services could wrap the metric.
func NewGaugeVec(opts prometheus.GaugeOpts, labels []string) GaugeVec {
	return WithSegmentGaugeVec(opts, prometheus.NewGaugeVec(opts, labels))

}

// NewGauge creates a GaugeVec, but with an empty list of labels (using With).
// When using NewGauge, it is perfectly safe (and preferred) to use a prometheus.Gauge type.
// As of writing this, only Segment is available. However, in the future, it is likely that many different services could wrap the metric.
func NewGauge(opts prometheus.GaugeOpts) prometheus.Gauge {
	return NewGaugeVec(opts, []string{}).With(prometheus.Labels{})
}

// NewCounterVec returns a CounterVec. This function uses options provided to the process to create
// the appropriate wrappers.
// As of writing this, only Segment is available. However, in the future, it is likely that many different services could wrap the metric.
func NewCounterVec(opts prometheus.CounterOpts, labels []string) CounterVec {
	return WithSegmentCounterVec(opts, prometheus.NewCounterVec(opts, labels))
}

// NewCounter creates a CounterVec, but with an empty list of labels (using With).
// When using NewCounter, it is perfectly safe (and preferred) to use a prometheus.Counter type.
// As of writing this, only Segment is available. However, in the future, it is likely that many different services could wrap the metric.
func NewCounter(opts prometheus.CounterOpts) prometheus.Counter {
	return NewCounterVec(opts, []string{}).With(prometheus.Labels{})
}

func isOptedOut(
	cmd *cobra.Command,
	l logrus.FieldLogger,
) bool {
	optOut, err := cmd.Flags().GetBool("sqa-opt-out")
	if !optOut {
		optOut, err = cmd.Flags().GetBool("disable-telemetry")
		if optOut {
			l.Warn(`Command line argument "--disable-telemetry" has been deprecated and will be removed in an upcoming release. Use "--sqa-opt-out" instead.`)
		}
	}
	cmdx.Must(err, "Unable to get command line flag.")

	if !optOut {
		optOut = viper.GetBool("sqa.opt_out")
	}

	if !optOut {
		optOut = viper.GetBool("DISABLE_TELEMETRY")
		if optOut {
			l.Warn(`Environment variable "DISABLE_TELEMETRY" has been deprecated and will be removed in an upcoming release. Use configuration key "sqa.opt_out: true" or environment variable "SQA_OPT_OUT=true" instead.`)
		}
	}

	return optOut
}

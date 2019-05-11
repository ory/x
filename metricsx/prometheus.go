package metricsx

import (
	"github.com/pborman/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/analytics-go"

	"github.com/ory/hydra/client"
	"github.com/ory/hydra/consent"
	"github.com/ory/hydra/driver"
	"github.com/ory/hydra/jwk"
	"github.com/ory/hydra/oauth2"
)

var (
	segmentOptions *metricsx.SegmentOptions
)

// GaugeVec is an interface which prometheus.GaugeVec implements.
// This allows for more generic wrapping of prometheus.GaugeVec
type GaugeVec interface {
	prometheus.Collector
	With(prometheus.Labels) prometheus.Gauge
	WithLabelValues(...string) prometheus.Gauge
}

// CounterVec is an interface which prometheus.CounterVec implements.
// This allows for more generic wrapping of prometheus.CounterVec
type CounterVec interface {
	prometheus.Collector
	With(prometheus.Labels) prometheus.Counter
	WithLabelValues(...string) prometheus.Counter
}

// NewGaugeVec returns a GaugeVec. This function uses options provided to the process to create
// the appropriate wrappers.
func NewGaugeVec(opts prometheus.GaugeOpts, labels []string) GaugeVec {
	var c GaugeVec = prometheus.NewGaugeVec(opts, labels)

	if segmentOptions != nil {
		c = WithSegmentGaugeVec(c)
	}

	return c
}

// NewCounterVec returns a CounterVec. This function uses options provided to the process to create
// the appropriate wrappers.
func NewCounterVec(opts prometheus.CounterOpts, labels []string) CounterVec {
	var c CounterVec = prometheus.NewCounterVec(opts, labels)

	if segmentOptions != nil {
		c = WithSegmentCounterVec(c)
	}

	return c
}

// Setup will initialize the available options
func Setup(d driver.Driver) {
	var (
		l   = d.Registry().Logger()
		ctx = context.Background()
	)

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

	if !optOut {
		l.Info("Software quality assurance features are enabled. Learn more at: https://www.ory.sh/docs/ecosystem/sqa")
		segmentOptions = &metricsx.SegmentOptions{
			Service: "ory-hydra",
			ClusterID: metricsx.Hash(fmt.Sprintf("%s|%s",
				d.Configuration().IssuerURL().String(),
				d.Configuration().DSN(),
			)),
			IsDevelopment: d.Configuration().DSN() == "memory" ||
				d.Configuration().IssuerURL().String() == "" ||
				strings.Contains(d.Configuration().IssuerURL().String(), "localhost"),
			WhitelistedPaths: []string{
				jwk.KeyHandlerPath,
				jwk.WellKnownKeysPath,

				client.ClientsHandlerPath,

				oauth2.DefaultConsentPath,
				oauth2.DefaultLoginPath,
				oauth2.DefaultPostLogoutPath,
				oauth2.DefaultLogoutPath,
				oauth2.DefaultErrorPath,
				oauth2.TokenPath,
				oauth2.AuthPath,
				oauth2.LogoutPath,
				oauth2.UserinfoPath,
				oauth2.WellKnownPath,
				oauth2.JWKPath,
				oauth2.IntrospectPath,
				oauth2.RevocationPath,
				oauth2.FlushPath,

				consent.ConsentPath,
				consent.ConsentPath + "/accept",
				consent.ConsentPath + "/reject",
				consent.LoginPath,
				consent.LoginPath + "/accept",
				consent.LoginPath + "/reject",
				consent.LogoutPath,
				consent.LogoutPath + "/accept",
				consent.LogoutPath + "/reject",
				consent.SessionsPath + "/login",
				consent.SessionsPath + "/consent",

				healthx.AliveCheckPath,
				healthx.ReadyCheckPath,
				healthx.VersionPath,
				driver.MetricsPrometheusPath,
				"/",
			},
			WriteKey:     "h8dRH3kVCWKkIFWydBmWsyYHR4M0u0vr",
			BuildVersion: d.Registry().BuildVersion(),
			BuildTime:    d.Registry().BuildDate(),
			BuildHash:    d.Registry().BuildHash(),
		}
	}
}

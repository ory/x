/*
 * Copyright © 2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @author		Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @copyright 	2015-2018 Aeneas Rekkas <aeneas+oss@aeneas.io>
 * @license 	Apache-2.0
 */

package metricsx

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ory/x/configx"

	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
	"github.com/ory/x/logrusx"
	"github.com/ory/x/resilience"

	"github.com/pborman/uuid"
	"github.com/urfave/negroni"

	analytics "github.com/ory/analytics-go/v4"
)

var instance *Service
var lock sync.Mutex

// Service helps with providing context on metrics.
type Service struct {
	optOut bool
	salt   string

	o       *Options
	context *analytics.Context

	c analytics.Client
	l *logrusx.Logger

	mem *MemoryStatistics
}

// Hash returns a hashed string of the value.
func Hash(value string) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(value))
	if err != nil {
		panic(fmt.Sprintf("unable to hash value"))
	}
	return hex.EncodeToString(hash.Sum(nil))
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

type void struct {
}

func (v *void) Logf(format string, args ...interface{}) {
}

func (v *void) Errorf(format string, args ...interface{}) {
}

// New returns a new metrics service. If one has been instantiated already, no new instance will be created.
func New(
	cmd *cobra.Command,
	l *logrusx.Logger,
	c *configx.Provider,
	o *Options,
) *Service {
	lock.Lock()
	defer lock.Unlock()

	if instance != nil {
		return instance
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
			Interval: time.Hour * 24,
		}
	}

	o.Config.Logger = new(void)

	if o.MemoryInterval < time.Minute {
		o.MemoryInterval = time.Hour * 12
	}

	segment, err := analytics.NewWithConfig(o.WriteKey, *o.Config)
	if err != nil {
		l.WithError(err).Fatalf("Unable to initialise software quality assurance features.")
		return nil
	}

	var oi analytics.OSInfo

	optOut, err := cmd.Flags().GetBool("sqa-opt-out")
	if err != nil {
		cmdx.Must(err, `Unable to get command line flag "sqa-opt-out": %s`, err)
	}

	if !optOut {
		optOut = c.Bool("sqa.opt_out")
	}

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

	instance = m

	go m.Identify()
	go m.ObserveMemory()

	return m
}

// Identify enables reporting to segment.
func (sw *Service) Identify() {
	if err := resilience.Retry(sw.l, time.Minute*5, time.Hour*24*30, func() error {
		return sw.c.Enqueue(analytics.Identify{
			UserId:  sw.o.ClusterID,
			Traits:  sw.context.Traits,
			Context: sw.context,
		})
	}); err != nil {
		sw.l.WithError(err).Debug("Could not commit anonymized environment information")
	}
}

// ObserveMemory commits memory statistics to segment.
func (sw *Service) ObserveMemory() {
	if sw.optOut {
		return
	}

	for {
		sw.mem.Update()
		if err := sw.c.Enqueue(analytics.Track{
			UserId:     sw.o.ClusterID,
			Event:      "memstats",
			Properties: analytics.Properties(sw.mem.ToMap()),
			Context:    sw.context,
		}); err != nil {
			sw.l.WithError(err).Debug("Could not commit anonymized telemetry data")
		}
		time.Sleep(sw.o.MemoryInterval)
	}
}

// ServeHTTP is a middleware for sending meta information to segment.
func (sw *Service) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	var start time.Time
	if !sw.optOut {
		start = time.Now()
	}

	next(rw, r)

	if sw.optOut {
		return
	}

	latency := time.Now().UTC().Sub(start.UTC()) / time.Millisecond

	scheme := "https:"
	if r.TLS == nil {
		scheme = "http:"
	}

	path := sw.anonymizePath(r.URL.Path, sw.salt)
	query := sw.anonymizeQuery(r.URL.Query(), sw.salt)

	// Collecting request info
	res := rw.(negroni.ResponseWriter)
	status := res.Status()
	size := res.Size()

	if err := sw.c.Enqueue(analytics.Page{
		UserId: sw.o.ClusterID,
		Name:   path,
		Properties: analytics.
			NewProperties().
			SetURL(scheme+"//"+sw.o.ClusterID+path+"?"+query).
			SetPath(path).
			SetName(path).
			Set("status", status).
			Set("size", size).
			Set("latency", latency).
			Set("method", r.Method),
		Context: sw.context,
	}); err != nil {
		sw.l.WithError(err).Debug("Could not commit anonymized telemetry data")
		// do nothing...
	}
}

func (sw *Service) Close() error {
	return sw.c.Close()
}

func (sw *Service) anonymizePath(path string, salt string) string {
	paths := sw.o.WhitelistedPaths
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

func (sw *Service) anonymizeQuery(query url.Values, salt string) string {
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

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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/segmentio/analytics-go"
	"github.com/urfave/negroni"
)

// Hash returns a hashed string of the value.
func Hash(value string) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(value))
	if err != nil {
		panic(fmt.Sprintf("unable to hash value"))
	}
	return hex.EncodeToString(hash.Sum(nil))
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

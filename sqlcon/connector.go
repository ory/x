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

// Package sqlcon provides helpers for dealing with SQL connectivity.
package sqlcon

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

func cleanURLQuery(in url.Values) (out url.Values) {
	out, _ = url.ParseQuery(in.Encode())
	out.Del("max_conns")
	out.Del("max_idle_conns")
	out.Del("max_conn_lifetime")
	out.Del("max_idle_conn_time")
	out.Del("parseTime")
	return out
}

// GetDriverName returns the driver name of a given DSN.
func GetDriverName(dsn string) string {
	return strings.Split(dsn, "://")[0]
}

func classifyDSN(dsn string) string {
	scheme := strings.Split(dsn, "://")[0]
	parts := strings.Split(dsn, "@")
	host := parts[len(parts)-1]
	return fmt.Sprintf("%s://*:*@%s", scheme, host)
}

func maxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}

func parseQuery(dsn string) (clean string, query url.Values, err error) {
	query = url.Values{}
	parts := strings.Split(dsn, "?")
	clean = parts[0]
	if len(parts) == 2 {
		if query, err = url.ParseQuery(parts[1]); err != nil {
			return "", query, errors.WithStack(err)
		}
	}
	return
}

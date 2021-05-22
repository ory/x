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

package sqlcon

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/dockertest/v3"
)

var (
	mysqlURL     string
	postgresURL  string
	cockroachURL string
	resources    []*dockertest.Resource
	lock         sync.RWMutex
)

func merge(u string, query url.Values) string {
	if strings.Contains(u, "?") {
		return u + "&" + query.Encode()
	}
	return u + "?" + query.Encode()
}

func TestClassifyDSN(t *testing.T) {
	for k, tc := range [][]string{
		{"mysql://foo:bar@tcp(baz:1234)/db?foo=bar", "mysql://*:*@tcp(baz:1234)/db?foo=bar"},
		{"mysql://foo@email.com:bar@tcp(baz:1234)/db?foo=bar", "mysql://*:*@tcp(baz:1234)/db?foo=bar"},
		{"postgres://foo:bar@baz:1234/db?foo=bar", "postgres://*:*@baz:1234/db?foo=bar"},
		{"postgres://foo@email.com:bar@baz:1234/db?foo=bar", "postgres://*:*@baz:1234/db?foo=bar"},
	} {
		t.Run(fmt.Sprintf("case=%d", k), func(t *testing.T) {
			assert.Equal(t, tc[1], classifyDSN(tc[0]))
		})
	}
}

func TestCleanQueryURL(t *testing.T) {
	a, err := url.ParseQuery("max_conn_lifetime=1h&max_idle_conn_time=1h&max_idle_conns=10&max_conns=10")
	require.NoError(t, err)

	b := cleanURLQuery(a)
	assert.NotEqual(t, a, b)
	assert.NotEqual(t, a.Encode(), b.Encode())
	assert.Equal(t, true, strings.Contains(a.Encode(), "max_idle_conn_time"))
	assert.Equal(t, false, strings.Contains(b.Encode(), "max_idle_conn_time"))
	assert.Equal(t, true, strings.Contains(a.Encode(), "max_conn_lifetime"))
	assert.Equal(t, false, strings.Contains(b.Encode(), "max_conn_lifetime"))
}

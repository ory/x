// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

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

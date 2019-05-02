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
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/dockertest"
	dockertestd "github.com/ory/x/sqlcon/dockertest"
)

var (
	mysqlURL     *url.URL
	postgresURL  *url.URL
	cockroachURL *url.URL
	resources    []*dockertest.Resource
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		dockertestd.Parallel([]func(){
			bootstrapMySQL,
			bootstrapPostgres,
			bootstrapCockroach,
		})
	}

	s := m.Run()
	killAll()
	os.Exit(s)
}

func merge(u *url.URL, params map[string]string) *url.URL {
	b := new(url.URL)
	*b = *u
	for k, v := range params {
		b.Query().Add(k, v)
	}
	return b
}

func TestDistributedTracing(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	databases := map[string]string{
		"mysql":     mysqlURL.String(),
		"postgres":  postgresURL.String(),
		"cockroach": cockroachURL.String(),
	}

	for driver, dsn := range databases {
		for _, testCase := range []struct {
			description   string
			sqlConnection *SQLConnection
		}{
			{
				description: fmt.Sprintf("%s: when tracing has been configured - spans should be created", driver),
				sqlConnection: mustSQL(t, dsn,
					WithDistributedTracing(),
					WithRandomDriverName(), // this test option is set to ensure a unique driver name is registered
					WithAllowRoot()),
			},
			{
				description: fmt.Sprintf("%s: no spans should be created if parent span is missing from context when `WithAllowRoot` has NOT been set", driver),
				sqlConnection: mustSQL(t, dsn,
					WithDistributedTracing(), // Notice that the WithAllowRoot() option has NOT been set
					WithRandomDriverName()),  // this test option is set to ensure a unique driver name is registered
			},
			{
				description:   fmt.Sprintf("%s: when tracing has NOT been configured - NO spans should be created", driver),
				sqlConnection: mustSQL(t, dsn), // Notice that the WithDistributedTracing() option has NOT been set
			},
			{
				description: fmt.Sprintf("%s: no arg tag should be added to spans if `WithOmitArgsFromTraceSpans` has been set", driver),
				sqlConnection: mustSQL(t, dsn,
					WithDistributedTracing(),
					WithRandomDriverName(), // this test option is set to ensure a unique driver name is registered
					WithAllowRoot(),
					WithOmitArgsFromTraceSpans()),
			},
		} {
			t.Run(fmt.Sprintf("case=%s", testCase.description), func(t *testing.T) {
				mockedTracer := mocktracer.New()
				defer mockedTracer.Reset()
				opentracing.SetGlobalTracer(mockedTracer)

				db, err := testCase.sqlConnection.GetDatabase()
				require.NoError(t, err)

				// Notice how no parent span exists in the provided context!
				db.QueryRowContext(context.TODO(), "SELECT NOW()")

				spans := mockedTracer.FinishedSpans()
				if testCase.sqlConnection.UseTracedDriver && testCase.sqlConnection.AllowRoot {
					assert.NotEmpty(t, spans)

					spansContainArgsTag := spansContainTag(spans, "args")
					if testCase.sqlConnection.OmitArgs {
						assert.False(t, spansContainArgsTag)
					} else {
						assert.True(t, spansContainArgsTag)
					}
				} else {
					assert.Empty(t, spans)
				}
			})
		}
	}
}

func TestRegisterDriver(t *testing.T) {
	for _, testCase := range []struct {
		description        string
		sqlConnection      *SQLConnection
		expectedDriverName string
		shouldError        bool
	}{
		{
			description:        "should return error if supplied DSN is unsupported for tracing",
			sqlConnection:      mustSQL(t, "unsupported://unsupported:secret@localhost:1337/mydb", WithDistributedTracing()),
			expectedDriverName: "",
			shouldError:        true,
		},
		{
			description:        "should return registered driver name if supplied DSN is valid for tracing",
			sqlConnection:      mustSQL(t, "mysql://foo@bar:baz@qux/db", WithDistributedTracing()),
			expectedDriverName: "instrumented-sql-driver",
			shouldError:        false,
		},
		{
			description:        "should return cockroach driver if a valid cockroach DSN is supplied",
			sqlConnection:      mustSQL(t, "cockroach://foo@bar:baz@qux/db"),
			expectedDriverName: "cockroach",
			shouldError:        false,
		},
	} {
		t.Run(fmt.Sprintf("case=%s", testCase.description), func(t *testing.T) {
			testCase.sqlConnection.L = logrus.New()
			driverName, err := testCase.sqlConnection.registerDriver()
			assert.Equal(t, testCase.expectedDriverName, driverName)
			if testCase.shouldError {
				assert.Error(t, err)
				assert.Empty(t, driverName)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, driverName)
			}
		})
	}
}

func TestCleanQueryURL(t *testing.T) {
	a, _ := url.Parse("mysql://foo:bar@baz/db?max_conn_lifetime=1h&max_idle_conns=10&max_conns=10")
	b := cleanURLQuery(a)
	assert.NotEqual(t, a, b)
	assert.NotEqual(t, a.String(), b.String())
	assert.Equal(t, true, strings.Contains(a.String(), "max_conn_lifetime"))
	assert.Equal(t, false, strings.Contains(b.String(), "max_conn_lifetime"))
}

func TestConnectionString(t *testing.T) {
	a, _ := url.Parse("mysql://foo@bar:baz@qux/db")
	b := connectionString(a)
	assert.NotEqual(t, b, a.String())
	assert.True(t, strings.HasPrefix(b, "foo%40bar:baz@qux"))
}

func mustSQL(t *testing.T, db string, opts ...OptionModifier) *SQLConnection {
	c, err := NewSQLConnection(db, logrus.New(), opts...)
	require.NoError(t, err)
	return c
}

func TestSQLConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode.")
		return
	}

	for _, tc := range []struct {
		s *SQLConnection
		d string
	}{
		{
			d: "mysql raw",
			s: mustSQL(t, mysqlURL.String()),
		},
		{
			d: "mysql max_conn_lifetime",
			s: mustSQL(t, merge(mysqlURL, map[string]string{"max_conn_lifetime": "1h"}).String()),
		},
		{
			d: "mysql max_conn_lifetime",
			s: mustSQL(t, merge(mysqlURL, map[string]string{"max_conn_lifetime": "1h", "max_idle_conns": "10", "max_conns": "10"}).String()),
		},
		{
			d: "pg raw",
			s: mustSQL(t, postgresURL.String()),
		},
		{
			d: "pg max_conn_lifetime",
			s: mustSQL(t, merge(postgresURL, map[string]string{"max_conn_lifetime": "1h"}).String()),
		},
		{
			d: "pg max_conn_lifetime",
			s: mustSQL(t, merge(postgresURL, map[string]string{"max_conn_lifetime": "1h", "max_idle_conns": "10", "max_conns": "10"}).String()),
		},
		{
			d: "crdb raw",
			s: mustSQL(t, cockroachURL.String()),
		},
		{
			d: "crdb max_conn_lifetime",
			s: mustSQL(t, merge(cockroachURL, map[string]string{"max_conn_lifetime": "1h"}).String()),
		},
		{
			d: "crdb max_conn_lifetime",
			s: mustSQL(t, merge(cockroachURL, map[string]string{"max_conn_lifetime": "1h", "max_idle_conns": "10", "max_conns": "10"}).String()),
		},
	} {
		t.Run(fmt.Sprintf("case=%s", tc.d), func(t *testing.T) {
			tc.s.L = logrus.New()
			db, err := tc.s.GetDatabase()
			require.NoError(t, err)

			require.Nil(t, db.Ping())

			// Test for parseTime support in MySQL
			tim := &time.Time{}
			require.Nil(t, db.QueryRow("SELECT NOW()").Scan(&tim))
		})
	}
}

func killAll() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not Connect to pool because %s", err)
	}

	for _, resource := range resources {
		if err := pool.Purge(resource); err != nil {
			log.Printf("Got an error while trying to purge resource: %s", err)
		}
	}

	resources = []*dockertest.Resource{}
}

func bootstrapMySQL() {
	if uu := os.Getenv("TEST_DATABASE_MYSQL"); uu != "" {
		log.Println("Found mysql test database config, skipping dockertest...")
		_, err := sqlx.Open("mysql", uu)
		if err != nil {
			log.Fatalf("Could not connect to bootstrapped database: %s", err)
		}
		u, _ := url.Parse(uu)
		mysqlURL = u
		return
	}

	pool, err := dockertest.NewPool("")
	pool.MaxWait = time.Minute * 5
	if err != nil {
		log.Fatalf("Could not Connect to docker: %s", err)
	}

	resource, err := pool.Run("mysql", "5.7", []string{"MYSQL_ROOT_PASSWORD=secret"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	urls := bootstrap("root:secret@(localhost:%s)/mysql?parseTime=true", "3306/tcp", "mysql", pool, resource)
	resources = append(resources, resource)
	u, _ := url.Parse("mysql://" + urls)
	mysqlURL = u
}

func bootstrapPostgres() {
	if uu := os.Getenv("TEST_DATABASE_POSTGRESQL"); uu != "" {
		log.Println("Found postgresql test database config, skipping dockertest...")
		_, err := sqlx.Open("postgres", uu)
		if err != nil {
			log.Fatalf("Could not connect to bootstrapped database: %s", err)
		}
		u, _ := url.Parse(uu)
		postgresURL = u
		return
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not Connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=hydra"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	urls := bootstrap("postgres://postgres:secret@localhost:%s/hydra?sslmode=disable", "5432/tcp", "postgres", pool, resource)
	resources = append(resources, resource)
	u, _ := url.Parse(urls)
	postgresURL = u
}

func bootstrapCockroach() {
	if uu := os.Getenv("TEST_DATABASE_COCKROACHDB"); uu != "" {
		log.Println("Found cockroachdb test database config, skipping dockertest...")
		_, err := sqlx.Open("postgres", uu)
		if err != nil {
			log.Fatalf("Could not connect to bootstrapped database: %s", err)
		}
		u, _ := url.Parse(uu)
		cockroachURL = u
		return
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not Connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "cockroachdb/cockroach",
		Tag:        "v2.1.6",
		Cmd:        []string{"start --insecure"},
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	urls := bootstrap("postgres://root@localhost:%s/defaultdb?sslmode=disable", "26257/tcp", "postgres", pool, resource)
	resources = append(resources, resource)
	u, _ := url.Parse(strings.Replace(urls, "postgres://", "cockroach://", 1))
	cockroachURL = u
}

func bootstrap(u, port, driver string, pool *dockertest.Pool, resource *dockertest.Resource) (urls string) {
	if err := pool.Retry(func() error {
		var err error
		urls = fmt.Sprintf(u, resource.GetPort(port))
		db, err := sqlx.Open(driver, urls)
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		pool.Purge(resource)
		log.Fatalf("Could not Connect to docker: %s", err)
	}
	resources = append(resources, resource)
	return
}

func spansContainTag(spans []*mocktracer.MockSpan, tagName string) bool {
	foundTag := false

	for _, span := range spans {
		if _, ok := span.Tags()[tagName]; ok {
			return true
		}
	}

	return foundTag
}

package dockertest

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ory/x/stringsx"

	"github.com/gobuffalo/pop/v6"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/ory/dockertest/v3"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/resilience"
)

// atexit := atexit.NewOnExit()
// atexit.Add(func() {
//	dockertest.KillAll()
// })
// atexit.Exit(testMain(m))

// func WrapCleanup

var resources = []*dockertest.Resource{}
var pool *dockertest.Pool

// KillAllTestDatabases deletes all test databases.
func KillAllTestDatabases() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	for _, r := range resources {
		if err := pool.Purge(r); err != nil {
			panic(err)
		}
	}

	resources = []*dockertest.Resource{}
}

// Register sets up OnExit.
func Register() *OnExit {
	onexit := NewOnExit()
	onexit.Add(func() {
		KillAllTestDatabases()
	})
	return onexit
}

// Parallel runs tasks in parallel.
func Parallel(fs []func()) {
	wg := sync.WaitGroup{}

	wg.Add(len(fs))
	for _, f := range fs {
		go func(ff func()) {
			defer wg.Done()
			ff()
		}(f)
	}

	wg.Wait()
}

func connect(dialect, driver, dsn string) (db *sqlx.DB, err error) {
	if scheme := strings.Split(dsn, "://")[0]; scheme == "mysql" {
		dsn = strings.Replace(dsn, "mysql://", "", -1)
	} else if scheme == "cockroach" {
		dsn = strings.Replace(dsn, "cockroach://", "postgres://", 1)
	}
	err = resilience.Retry(
		logrusx.New("", ""),
		time.Second*5,
		time.Minute*5,
		func() (err error) {
			db, err = sqlx.Open(dialect, dsn)
			if err != nil {
				log.Printf("Connecting to database %s failed: %s", driver, err)
				return err
			}

			if err := db.Ping(); err != nil {
				log.Printf("Pinging database %s failed: %s", driver, err)
				return err
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Errorf("Unable to connect to %s (%s): %s", driver, dsn, err)
	}
	log.Printf("Connected to database %s", driver)
	return db, nil
}

func connectPop(t require.TestingT, url string) (c *pop.Connection) {
	require.NoError(t, resilience.Retry(logrusx.New("", ""), time.Second*5, time.Minute*5, func() error {
		var err error
		c, err = pop.NewConnection(&pop.ConnectionDetails{
			URL: url,
		})
		if err != nil {
			log.Printf("could not create pop connection")
			return err
		}
		if err := c.Open(); err != nil {
			// an Open error probably means we have a problem with the connections config
			log.Printf("could not open pop connection: %+v", err)
			return err
		}
		return c.RawQuery("select version()").Exec()
	}))
	return
}

// ## PostgreSQL ##

func startPostgreSQL() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("postgres", "11.8", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=postgres"})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunTestPostgreSQL runs a PostgreSQL database and returns the URL to it.
func RunTestPostgreSQL(t testing.TB) string {
	if dsn := os.Getenv("TEST_DATABASE_POSTGRESQL"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_POSTGRESQL is set to: %s", dsn)
		return dsn
	}

	u, err := RunPostgreSQL()
	require.NoError(t, err)

	return u
}

// RunPostgreSQL runs a PostgreSQL database and returns the URL to it.
func RunPostgreSQL() (string, error) {
	resource, err := startPostgreSQL()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("postgres://postgres:secret@127.0.0.1:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp")), nil
}

// ConnectToTestPostgreSQL connects to a PostgreSQL database.
func ConnectToTestPostgreSQL() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_POSTGRESQL"); dsn != "" {
		return connect("pgx", "postgres", dsn)
	}

	resource, err := startPostgreSQL()
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("postgres://postgres:secret@localhost:%s/postgres?sslmode=disable", "5432/tcp", "pgx", pool, resource)
	return db, nil
}

func ConnectToTestPostgreSQLPop(t testing.TB) *pop.Connection {
	url := RunTestPostgreSQL(t)
	return connectPop(t, url)
}

// ## MySQL ##

func startMySQL() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("mysql", "8.0", []string{"MYSQL_ROOT_PASSWORD=secret"})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunMySQL runs a RunMySQL database and returns the URL to it.
func RunMySQL() (string, error) {
	resource, err := startMySQL()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("mysql://root:secret@(localhost:%s)/mysql?parseTime=true&multiStatements=true", resource.GetPort("3306/tcp")), nil
}

// RunTestMySQL runs a MySQL database and returns the URL to it.
func RunTestMySQL(t testing.TB) string {
	if dsn := os.Getenv("TEST_DATABASE_MYSQL"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_MYSQL is set to: %s", dsn)
		return dsn
	}

	u, err := RunMySQL()
	require.NoError(t, err)

	return u
}

// ConnectToTestMySQL connects to a MySQL database.
func ConnectToTestMySQL() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_MYSQL"); dsn != "" {
		log.Println("Found mysql test database config, skipping dockertest...")
		return connect("mysql", "mysql", dsn)
	}

	resource, err := startMySQL()
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("root:secret@(localhost:%s)/mysql?parseTime=true", "3306/tcp", "mysql", pool, resource)
	return db, nil
}

func ConnectToTestMySQLPop(t testing.TB) *pop.Connection {
	url := RunTestMySQL(t)
	return connectPop(t, url)
}

// ## CockroachDB

func startCockroachDB(version string) (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "cockroachdb/cockroach",
		Tag:        stringsx.Coalesce(version, "v20.2.5"),
		Cmd:        []string{"start-single-node", "--insecure"},
	})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunCockroachDB runs a CockroachDB database and returns the URL to it.
func RunCockroachDB() (string, error) {
	return RunCockroachDBWithVersion("")
}

// RunCockroachDB runs a CockroachDB database and returns the URL to it.
func RunCockroachDBWithVersion(version string) (string, error) {
	resource, err := startCockroachDB(version)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("cockroach://root@localhost:%s/defaultdb?sslmode=disable", resource.GetPort("26257/tcp")), nil
}

// RunTestCockroachDB runs a CockroachDB database and returns the URL to it.
func RunTestCockroachDB(t testing.TB) string {
	return RunTestCockroachDBWithVersion(t, "")
}

// RunTestCockroachDB runs a CockroachDB database and returns the URL to it.
func RunTestCockroachDBWithVersion(t testing.TB, version string) string {
	if dsn := os.Getenv("TEST_DATABASE_COCKROACHDB"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_COCKROACHDB is set to: %s", dsn)
		return dsn
	}

	u, err := RunCockroachDBWithVersion(version)
	require.NoError(t, err)

	return u
}

// ConnectToTestCockroachDB connects to a CockroachDB database.
func ConnectToTestCockroachDB() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_COCKROACHDB"); dsn != "" {
		log.Println("Found cockroachdb test database config, skipping dockertest...")
		return connect("pgx", "cockroach", dsn)
	}

	resource, err := startCockroachDB("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("postgres://root@localhost:%s/defaultdb?sslmode=disable", "26257/tcp", "pgx", pool, resource)
	return db, nil
}

func ConnectToTestCockroachDBPop(t testing.TB) *pop.Connection {
	url := RunTestCockroachDB(t)
	return connectPop(t, url)
}

func bootstrap(u, port, d string, pool *dockertest.Pool, resource *dockertest.Resource) (db *sqlx.DB) {
	if err := resilience.Retry(logrusx.New("", ""), time.Second*5, time.Minute*5, func() error {
		var err error
		db, err = sqlx.Open(d, fmt.Sprintf(u, resource.GetPort(port)))
		if err != nil {
			return err
		}

		return db.Ping()
	}); err != nil {
		if pErr := pool.Purge(resource); pErr != nil {
			log.Fatalf("Could not connect to docker and unable to remove image: %s - %s", err, pErr)
		}
		log.Fatalf("Could not connect to docker: %s", err)
	}
	return
}

var comments = regexp.MustCompile("(--[^\n]*\n)|(?s:/\\*.+\\*/)")

func StripDump(d string) string {
	d = comments.ReplaceAllLiteralString(d, "")
	d = strings.TrimPrefix(d, "Command \"dump\" is deprecated, cockroach dump will be removed in a subsequent release.\r\nFor details, see: https://github.com/cockroachdb/cockroach/issues/54040\r\n")
	d = strings.ReplaceAll(d, "\r\n", "")
	d = strings.ReplaceAll(d, "\t", " ")
	d = strings.ReplaceAll(d, "\n", " ")
	return d
}

func DumpSchema(ctx context.Context, t *testing.T, db string) string {
	var containerPort string
	var cmd []string

	switch c := stringsx.SwitchExact(db); {
	case c.AddCase("postgres"):
		containerPort = "5432"
		cmd = []string{"pg_dump", "-U", "postgres", "-s", "-T", "hydra_*_migration", "-T", "schema_migration"}
	case c.AddCase("mysql"):
		containerPort = "3306"
		cmd = []string{"/usr/bin/mysqldump", "-u", "root", "--password=secret", "mysql"}
	case c.AddCase("cockroach"):
		containerPort = "26257"
		cmd = []string{"./cockroach", "dump", "defaultdb", "--insecure", "--dump-mode=schema"}
	default:
		t.Log(c.ToUnknownCaseErr())
		t.FailNow()
		return ""
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		Quiet:   true,
		Filters: filters.NewArgs(filters.Arg("expose", containerPort)),
	})
	require.NoError(t, err)

	if len(containers) != 1 {
		t.Logf("Ambiguous amount of %s containers: %d", db, len(containers))
		t.FailNow()
	}

	process, err := cli.ContainerExecCreate(ctx, containers[0].ID, types.ExecConfig{
		Tty:          true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	require.NoError(t, err)

	resp, err := cli.ContainerExecAttach(ctx, process.ID, types.ExecStartCheck{
		Tty: true,
	})
	require.NoError(t, err)
	dump, err := ioutil.ReadAll(resp.Reader)
	require.NoError(t, err, "%s", dump)

	return StripDump(string(dump))
}

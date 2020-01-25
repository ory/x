package dockertest

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ory/dockertest/v3"

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
		logrus.New(),
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

// ## PostgreSQL ##

func startPostgreSQL() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=postgres"})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunTestPostgreSQL runs a PostgreSQL database and returns the URL to it.
func RunTestPostgreSQL(t *testing.T) string {
	if dsn := os.Getenv("TEST_DATABASE_POSTGRESQL"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_POSTGRESQL is set to: %s", dsn)
		return dsn
	}

	resource, err := startPostgreSQL()
	require.NoError(t, err)

	return fmt.Sprintf("postgres://postgres:secret@127.0.0.1:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))
}

// ConnectToTestPostgreSQL connects to a PostgreSQL database.
func ConnectToTestPostgreSQL() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_POSTGRESQL"); dsn != "" {
		return connect("postgres", "postgres", dsn)
	}

	resource, err := startPostgreSQL()
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("postgres://postgres:secret@localhost:%s/postgres?sslmode=disable", "5432/tcp", "postgres", pool, resource)
	return db, nil
}

// ## MySQL ##

func startMySQL() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("mysql", "5.7", []string{"MYSQL_ROOT_PASSWORD=secret"})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunTestMySQL runs a MySQL database and returns the URL to it.
func RunTestMySQL(t *testing.T) string {
	if dsn := os.Getenv("TEST_DATABASE_MYSQL"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_MYSQL is set to: %s", dsn)
		return dsn
	}

	resource, err := startMySQL()
	require.NoError(t, err)

	return fmt.Sprintf("mysql://root:secret@(localhost:%s)/mysql?parseTime=true&multiStatements=true", resource.GetPort("3306/tcp"))
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

// ## CockroachDB

func startCockroachDB() (*dockertest.Resource, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "cockroachdb/cockroach",
		Tag:        "v19.2.0",
		Cmd:        []string{"start", "--insecure"},
	})
	if err == nil {
		resources = append(resources, resource)
	}
	return resource, err
}

// RunTestCockroachDB runs a CockroachDB database and returns the URL to it.
func RunTestCockroachDB(t *testing.T) string {
	if dsn := os.Getenv("TEST_DATABASE_COCKROACHDB"); dsn != "" {
		t.Logf("Skipping Docker setup because environment variable TEST_DATABASE_COCKROACHDB is set to: %s", dsn)
		return dsn
	}

	resource, err := startCockroachDB()
	require.NoError(t, err)

	return fmt.Sprintf("cockroach://root@localhost:%s/defaultdb?sslmode=disable", resource.GetPort("26257/tcp"))
}

// ConnectToTestCockroachDB connects to a CockroachDB database.
func ConnectToTestCockroachDB() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_COCKROACHDB"); dsn != "" {
		log.Println("Found cockroachdb test database config, skipping dockertest...")
		return connect("postgres", "cockroach", dsn)
	}

	resource, err := startCockroachDB()
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("postgres://root@localhost:%s/defaultdb?sslmode=disable", "26257/tcp", "postgres", pool, resource)
	return db, nil
}

func bootstrap(u, port, d string, pool *dockertest.Pool, resource *dockertest.Resource) (db *sqlx.DB) {
	if err := resilience.Retry(logrus.New(), time.Second*5, time.Minute*5, func() error {
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

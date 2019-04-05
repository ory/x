package dockertest

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ory/dockertest"
	"github.com/ory/x/resilience"
)

//atexit := atexit.NewOnExit()
//atexit.Add(func() {
//	dockertest.KillAll()
//})
//atexit.Exit(testMain(m))

//func WrapCleanup

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
		go func(f func()) {
			defer wg.Done()
			f()
		}(f)
	}

	wg.Wait()
}

func connect(dialect, driver, dsn string) (db *sqlx.DB, err error) {
	clean, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	if clean.Scheme == "mysql" {
		dsn = strings.Replace(dsn, "mysql://", "", -1)
	}
	if clean.Scheme == "cockroach" {
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

// ConnectToTestPostgreSQL connects to a PostgreSQL database.
func ConnectToTestPostgreSQL() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_POSTGRESQL"); dsn != "" {
		log.Println("Found postgresql test database config, skipping dockertest...")
		return connect("postgres", "postgres", dsn)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=secret", "POSTGRES_DB=hydra"})
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("postgres://postgres:secret@localhost:%s/hydra?sslmode=disable", "5432/tcp", "postgres", pool, resource)
	return db, nil
}

// ConnectToTestMySQL connects to a MySQL database.
func ConnectToTestMySQL() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_MYSQL"); dsn != "" {
		log.Println("Found mysql test database config, skipping dockertest...")
		return connect("mysql", "mysql", dsn)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.Run("mysql", "5.7", []string{"MYSQL_ROOT_PASSWORD=secret"})
	if err != nil {
		return nil, errors.Wrap(err, "Could not start resource")
	}

	db := bootstrap("root:secret@(localhost:%s)/mysql?parseTime=true", "3306/tcp", "mysql", pool, resource)
	return db, nil
}

// ConnectToTestCockroachDB connects to a CockroachDB database.
func ConnectToTestCockroachDB() (*sqlx.DB, error) {
	if dsn := os.Getenv("TEST_DATABASE_COCKROACHDB"); dsn != "" {
		log.Println("Found cockroachdb test database config, skipping dockertest...")
		return connect("postgres", "cockroach", dsn)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, errors.Wrap(err, "Could not connect to docker")
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "cockroachdb/cockroach",
		Tag:        "v2.1.6",
		Cmd:        []string{"start --insecure"},
	})
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
	resources = append(resources, resource)
	return
}

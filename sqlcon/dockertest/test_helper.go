package dockertest

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/ory/dockertest"
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
			f()
			wg.Done()
		}(f)
	}

	wg.Wait()
}

// ConnectToTestPostgreSQL connects to a PostgreSQL database.
func ConnectToTestPostgreSQL() (*sqlx.DB, error) {
	if url := os.Getenv("TEST_DATABASE_POSTGRESQL"); url != "" {
		log.Println("Found postgresql test database config, skipping dockertest...")
		db, err := sqlx.Open("postgres", url)
		if err != nil {
			return nil, errors.Wrap(err, "Could not connect to bootstrapped database")
		}
		return db, nil
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
	if url := os.Getenv("TEST_DATABASE_MYSQL"); url != "" {
		log.Println("Found mysql test database config, skipping dockertest...")
		db, err := sqlx.Open("mysql", url)
		if err != nil {
			return nil, errors.Wrap(err, "Could not connect to bootstrapped database")
		}
		return db, nil
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

func bootstrap(u, port, d string, pool *dockertest.Pool, resource *dockertest.Resource) (db *sqlx.DB) {
	if err := pool.Retry(func() error {
		var err error
		db, err = sqlx.Open(d, fmt.Sprintf(u, resource.GetPort(port)))
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

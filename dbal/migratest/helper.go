package migratest

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pborman/uuid"
	"github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/sqlcon/dockertest"
)

// RunPackrMigrationTests runs migration tests from packr migrations.
func RunPackrMigrationTests(
	t *testing.T, schema, data map[string]*migrate.PackrMigrationSource,
	init, cleanup func(*testing.T, *sqlx.DB),
	runner func(*testing.T, *sqlx.DB, int),
) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	var n = -1
	for _, s := range schema {
		if n == -1 {
			n = len(s.Box.List())
		}
		require.Equal(t, n, len(s.Box.List()))
	}

	var m sync.Mutex
	var dbs = map[string]*sqlx.DB{}
	var mid = uuid.New()

	dockertest.Parallel([]func(){
		func() {
			db, err := dockertest.ConnectToTestPostgreSQL()
			if err != nil {
				t.Fatalf("Could not connect to database: %v", err)
			}
			m.Lock()
			dbs["postgres"] = db
			m.Unlock()
		},
		func() {
			db, err := dockertest.ConnectToTestMySQL()
			if err != nil {
				t.Fatalf("Could not connect to database: %v", err)
			}
			m.Lock()
			dbs["mysql"] = db
			m.Unlock()
		},
	})

	for name, db := range dbs {
		t.Run(fmt.Sprintf("database=%s", name), func(t *testing.T) {
			init(t, db)

			for step := 0; step < n; step++ {
				t.Run(fmt.Sprintf("up=%d", step), func(t *testing.T) {
					migrate.SetTable(mid)
					n, err := migrate.ExecMax(db.DB, db.DriverName(), schema[name], migrate.Up, 1)
					require.NoError(t, err)
					require.Equal(t, n, 1)

					migrate.SetTable(mid + "_data")
					n, err = migrate.ExecMax(db.DB, db.DriverName(), data[name], migrate.Up, 1)
					require.NoError(t, err)
					require.Equal(t, n, 1)
				})
			}

			for step := 0; step < n; step++ {
				runner(t, db, step)
			}

			migrate.SetTable(mid)
			for step := 0; step < n; step++ {
				t.Run(fmt.Sprintf("down=%d", step), func(t *testing.T) {
					n, err := migrate.ExecMax(db.DB, db.DriverName(), schema[name], migrate.Down, 1)
					require.NoError(t, err)
					require.Equal(t, n, 1)
				})
			}

			cleanup(t, db)
		})
	}
}

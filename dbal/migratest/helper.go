package migratest

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ory/x/dbal"

	"github.com/jmoiron/sqlx"
	"github.com/pborman/uuid"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/sqlcon/dockertest"
)

// MigrationSchemas contains several schemas.
type MigrationSchemas []map[string]*dbal.PackrMigrationSource

// RunPackrMigrationTests runs migration tests from packr migrations.
func RunPackrMigrationTests(
	t *testing.T, schema, data MigrationSchemas,
	init, cleanup func(*testing.T, *sqlx.DB),
	runner func(*testing.T, *sqlx.DB, int, int, int),
) {
	if testing.Short() {
		t.SkipNow()
		return
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

	if data != nil {
		require.Equal(t, len(schema), len(data))
	}

	for name, db := range dbs {
		t.Run(fmt.Sprintf("database=%s", name), func(t *testing.T) {
			init(t, db)

			for sk, ss := range schema {
				t.Run(fmt.Sprintf("schema=%d/run", sk), func(t *testing.T) {
					steps := len(ss[name].Box.List())
					for step := 0; step < steps; step++ {
						t.Run(fmt.Sprintf("up=%d", step), func(t *testing.T) {
							migrate.SetTable(fmt.Sprintf("%s_%d", mid, sk))
							n, err := migrate.ExecMax(db.DB, db.DriverName(), ss[name], migrate.Up, 1)
							require.NoError(t, err)
							require.Equal(t, n, 1, sk)

							t.Run(fmt.Sprintf("data=%d", step), func(t *testing.T) {
								if data == nil || data[sk] == nil {
									t.Skip("Skipping data creation because no schema specified...")
									return
								}

								migrate.SetTable(fmt.Sprintf("%s_%d_data", mid, sk))
								n, err = migrate.ExecMax(db.DB, db.DriverName(), data[sk][name], migrate.Up, 1)
								require.NoError(t, err)
								require.Equal(t, n, 1)
							})
						})
					}

					for step := 0; step < steps; step++ {
						t.Run(fmt.Sprintf("runner=%d", step), func(t *testing.T) {
							runner(t, db, sk, step, steps)
						})
					}
				})
			}

			for sk := len(schema) - 1; sk >= 0; sk-- {
				ss := schema[sk]

				t.Run(fmt.Sprintf("schema=%d/cleanup", sk), func(t *testing.T) {
					steps := len(ss[name].Box.List())

					migrate.SetTable(fmt.Sprintf("%s_%d", mid, sk))
					for step := 0; step < steps; step++ {
						t.Run(fmt.Sprintf("down=%d", step), func(t *testing.T) {
							n, err := migrate.ExecMax(db.DB, db.DriverName(), ss[name], migrate.Down, 1)
							require.NoError(t, err)
							require.Equal(t, n, 1)
						})
					}
				})
			}

			cleanup(t, db)
		})
	}
}

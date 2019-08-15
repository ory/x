package migratest

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/dbal"
	"github.com/ory/x/sqlcon/dockertest"
)

// MigrationSchemas contains several schemas.
type MigrationSchemas []map[string]*dbal.PackrMigrationSource

// RunPackrMigrationTests runs migration tests from packr migrations.
func RunPackrMigrationTests(
	t *testing.T, schema, data MigrationSchemas,
	init, cleanup func(*testing.T, *sqlx.DB),
	runner func(*testing.T, string, *sqlx.DB, int, int, int),
) {
	if testing.Short() {
		t.SkipNow()
		return
	}

	var m sync.Mutex
	var dbs = map[string]*sqlx.DB{}
	var mid = uuid.New().String()

	var dbnames = map[string]bool{}
	for _, ms := range schema {
		for dbname := range ms {
			dbnames[dbname] = true
		}
	}

	var connectors []func()
	for dbname := range dbnames {
		switch dbname {
		case dbal.DriverPostgreSQL:
			connectors = append(connectors, func() {
				db, err := dockertest.ConnectToTestPostgreSQL()
				if err != nil {
					t.Fatalf("Could not connect to database: %v", err)
				}
				m.Lock()
				dbs[dbal.DriverPostgreSQL] = db
				m.Unlock()
			})
		case dbal.DriverMySQL:
			connectors = append(connectors, func() {
				db, err := dockertest.ConnectToTestMySQL()
				if err != nil {
					t.Fatalf("Could not connect to database: %v", err)
				}
				m.Lock()
				dbs[dbal.DriverMySQL] = db
				m.Unlock()
			})
		case dbal.DriverCockroachDB:
			connectors = append(connectors, func() {
				db, err := dockertest.ConnectToTestCockroachDB()
				if err != nil {
					t.Fatalf("Could not connect to database: %v", err)
				}
				m.Lock()
				dbs[dbal.DriverCockroachDB] = db
				m.Unlock()
			})
		default:
			panic(fmt.Sprintf("Database name %s unknown", dbname))
		}
	}

	dockertest.Parallel(connectors)

	if data != nil {
		require.Equal(t, len(schema), len(data))
	}

	for name, db := range dbs {
		dialect := db.DriverName()
		t.Run(fmt.Sprintf("database=%s", name), func(t *testing.T) {
			init(t, db)

			for sk, ss := range schema {
				t.Run(fmt.Sprintf("schema=%d/run", sk), func(t *testing.T) {
					steps := len(ss[name].Box.List())
					for step := 0; step < steps; step++ {
						t.Run(fmt.Sprintf("up=%d", step), func(t *testing.T) {
							migrate.SetTable(fmt.Sprintf("%s_%d", mid, sk))
							n, err := migrate.ExecMax(db.DB, dialect, ss[name], migrate.Up, 1)
							require.NoError(t, err)
							require.Equal(t, n, 1, sk)

							t.Run(fmt.Sprintf("data=%d", step), func(t *testing.T) {
								if data == nil || data[sk] == nil {
									t.Skip("Skipping data creation because no schema specified...")
									return
								}

								migrate.SetTable(fmt.Sprintf("%s_%d_data", mid, sk))
								n, err = migrate.ExecMax(db.DB, dialect, data[sk][name], migrate.Up, 1)
								require.NoError(t, err)
								require.Equal(t, 1, n)
							})
						})
					}

					for step := 0; step < steps; step++ {
						t.Run(fmt.Sprintf("runner=%d", step), func(t *testing.T) {
							runner(t, name, db, sk, step, steps)
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
							n, err := migrate.ExecMax(db.DB, dialect, ss[name], migrate.Down, 1)
							require.NoError(t, err)
							require.Equal(t, 1, n)
						})
					}
				})
			}

			cleanup(t, db)
		})
	}
}

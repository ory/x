package popx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/gobuffalo/pop/v5"

	"github.com/ory/x/logrusx"

	"github.com/pkg/errors"
)

var mrx = regexp.MustCompile(`^(\d+)_([^.]+)(\.[a-z0-9]+)?\.(up|down)\.(sql|fizz)$`)

// NewMigrator returns a new "blank" migrator. It is recommended
// to use something like MigrationBox or FileMigrator. A "blank"
// Migrator should only be used as the basis for a new type of
// migration system.
func NewMigrator(c *pop.Connection, l *logrusx.Logger) Migrator {
	return Migrator{
		Connection: c,
		l:          l,
		Migrations: map[string]pop.Migrations{
			"up":   {},
			"down": {},
		},
	}
}

// Migrator forms the basis of all migrations systems.
// It does the actual heavy lifting of running migrations.
// When building a new migration system, you should embed this
// type into your migrator.
type Migrator struct {
	Connection *pop.Connection
	SchemaPath string
	Migrations map[string]pop.Migrations
	l          *logrusx.Logger
}

func (m Migrator) migrationIsCompatible(dialect string, mi pop.Migration) bool {
	if mi.DBType == "all" || mi.DBType == dialect {
		return true
	}
	return false
}

// Up runs pending "up" migrations and applies them to the database.
func (m Migrator) Up() error {
	_, err := m.UpTo(0)
	return err
}

// UpTo runs up to step "up" migrations and applies them to the database.
// If step <= 0 all pending migrations are run.
func (m Migrator) UpTo(step int) (applied int, err error) {
	c := m.Connection
	err = m.exec(func() error {
		mtn := c.MigrationTableName()
		mfs := m.Migrations["up"]
		mfs.Filter(func(mf pop.Migration) bool {
			return m.migrationIsCompatible(c.Dialect.Name(), mf)
		})
		sort.Sort(mfs)
		for _, mi := range mfs {
			exists, err := c.Where("version = ?", mi.Version).Exists(mtn)
			if err != nil {
				return errors.Wrapf(err, "problem checking for migration version %s", mi.Version)
			}

			if exists {
				m.l.WithField("version", mi.Version).Debug("Migration has already been applied, skipping.")
				continue
			}

			if len(mi.Version) > 14 {
				m.l.WithField("version", mi.Version).Debug("Migration has not been applied but it might be a legacy migration, investigating.")

				legacyVersion := mi.Version[:14]
				exists, err = c.Where("version = ?", legacyVersion).Exists(mtn)
				if err != nil {
					return errors.Wrapf(err, "problem checking for migration version %s", mi.Version)
				}

				if exists {
					m.l.WithField("version", mi.Version).WithField("legacy_version", legacyVersion).WithField("migration_table", mtn).Debug("Migration has already been applied in a legacy migration run. Updating version in migration table.")
					if err := c.Transaction(func(tx *pop.Connection) error {
						// We do not want to remove the legacy migration version or subsequent migrations might be applied twice.
						//
						// Do not activate the following - it is just for reference.
						//
						// if _, err := tx.Store.Exec(fmt.Sprintf("DELETE FROM %s WHERE version = ?", mtn), legacyVersion); err != nil {
						//	return errors.Wrapf(err, "problem removing legacy version %s", mi.Version)
						// }

						return errors.Wrapf(tx.RawQuery(fmt.Sprintf("INSERT INTO %s (version) VALUES ('%s')", mtn, mi.Version)).Exec(), "problem inserting migration version %s", mi.Version)
					}); err != nil {
						return err
					}
					continue
				}
			}

			m.l.WithField("version", mi.Version).Debug("Migration has not yet been applied, running migration.")
			err = c.Transaction(func(tx *pop.Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return err
				}
				return errors.Wrapf(tx.RawQuery(fmt.Sprintf("INSERT INTO %s (VERSION) VALUES ('%s')", mtn, mi.Version)).Exec(), "problem inserting migration version %s", mi.Version)
			})
			if err != nil {
				return err
			}
			m.l.Debugf("> %s", mi.Name)
			applied++
			if step > 0 && applied >= step {
				break
			}
		}
		if applied == 0 {
			m.l.Debugf("Migrations already up to date, nothing to apply")
		} else {
			m.l.Debugf("Successfully applied %d migrations.", applied)
		}
		return nil
	})
	return
}

// Down runs pending "down" migrations and rolls back the
// database by the specified number of steps.
func (m Migrator) Down(step int) error {
	c := m.Connection
	return m.exec(func() error {
		mtn := c.MigrationTableName()
		count, err := c.Count(mtn)
		if err != nil {
			return errors.Wrap(err, "migration down: unable count existing migration")
		}
		mfs := m.Migrations["down"]
		mfs.Filter(func(mf pop.Migration) bool {
			return m.migrationIsCompatible(c.Dialect.Name(), mf)
		})
		sort.Sort(sort.Reverse(mfs))
		// skip all ran migration
		if len(mfs) > count {
			mfs = mfs[len(mfs)-count:]
		}
		// run only required steps
		if step > 0 && len(mfs) >= step {
			mfs = mfs[:step]
		}
		for _, mi := range mfs {
			exists, err := c.Where("version = ?", mi.Version).Exists(mtn)
			if err != nil {
				return errors.Wrapf(err, "problem checking for migration version %s", mi.Version)
			}

			if !exists && len(mi.Version) > 14 {
				legacyVersion := mi.Version[:14]
				legacyVersionExists, err := c.Where("version = ?", legacyVersion).Exists(mtn)
				if err != nil {
					return errors.Wrapf(err, "problem checking for migration version %s", mi.Version)
				}

				if !legacyVersionExists {
					return errors.Wrapf(err, "problem checking for migration version %s", legacyVersion)
				}
			} else if !exists {
				return errors.Errorf("migration version %s does not exist", mi.Version)
			}

			err = c.Transaction(func(tx *pop.Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return err
				}
				err = tx.RawQuery(fmt.Sprintf("DELETE FROM %s WHERE VERSION = ?", mtn), mi.Version).Exec()
				return errors.Wrapf(err, "problem deleting migration version %s", mi.Version)
			})
			if err != nil {
				return err
			}

			m.l.Debugf("< %s", mi.Name)
		}
		return nil
	})
}

// Reset the database by running the down migrations followed by the up migrations.
func (m Migrator) Reset() error {
	err := m.Down(-1)
	if err != nil {
		return err
	}
	return m.Up()
}

// CreateSchemaMigrations sets up a table to track migrations. This is an idempotent
// operation.
func CreateSchemaMigrations(c *pop.Connection, l *logrusx.Logger) error {
	mtn := c.MigrationTableName()
	err := c.Open()
	if err != nil {
		return errors.Wrap(err, "could not open connection")
	}

	l.WithField("migration_table", mtn).Debug("Checking if legacy migration table exists.")
	_, err = c.Store.Exec(fmt.Sprintf("select version from %s", mtn))
	if err != nil {
		l.WithError(err).WithField("migration_table", mtn).Debug("An error occurred while checking for the legacy migration table, maybe it does not exist yet? Trying to create.")

		// This means that the legacy pop migrator has not yet been applied
		if err := c.Transaction(func(tx *pop.Connection) error {
			if err := tx.RawQuery(fmt.Sprintf(`CREATE TABLE "%s" ("version" VARCHAR (14) NOT NULL);`, mtn)).Exec(); err != nil {
				return errors.WithStack(err)
			}

			return errors.WithStack(tx.RawQuery(fmt.Sprintf(`CREATE UNIQUE INDEX "schema_migration_version_idx" ON "%s" (version);`, mtn)).Exec())
		}); err != nil {
			return err
		}

		l.WithError(err).WithField("migration_table", mtn).Debug("Legacy migration table created successfully.")
	}

	l.WithField("migration_table", mtn).Debug("Checking if transactional migration table exists.")
	_, err = c.Store.Exec(fmt.Sprintf("select version, version_self from %s", mtn))
	if err != nil {
		l.WithError(err).WithField("migration_table", mtn).Debug("An error occurred while checking for the transactional migration table, maybe it does not exist yet? Trying to create.")
		// This means the new pop migrator has also not yet been applied, do that now.

		withOn := fmt.Sprintf(" ON %s", mtn)
		if c.Dialect.Name() != "mysql" {
			withOn = ""
		}
		workload := [][]string{
			{
				fmt.Sprintf(`DROP INDEX %s_version_idx%s`, mtn, withOn),
				fmt.Sprintf(`ALTER TABLE %s RENAME TO %s_pop_legacy`, mtn, mtn),
			},
			{
				fmt.Sprintf(`CREATE TABLE %s (version VARCHAR (48) NOT NULL, version_self INT NOT NULL DEFAULT 0)`, mtn),
				fmt.Sprintf(`CREATE UNIQUE INDEX %s_version_idx ON %s (version)`, mtn, mtn),
			},
			{
				fmt.Sprintf(`INSERT INTO %s (version) SELECT version FROM %s_pop_legacy`, mtn, mtn),
			},
		}

		for _, statements := range workload {
			// This means that the legacy pop migrator has not yet been applied
			if err := c.Transaction(func(tx *pop.Connection) error {
				for _, statement := range statements {
					if err := tx.RawQuery(statement).Exec(); err != nil {
						return errors.Wrapf(err, "unable to execute statement: %s", statement)
					}
				}

				return nil
			}); err != nil {
				return err
			}
		}

		l.WithError(err).WithField("migration_table", mtn).Debug("Transactional migration table created successfully.")
		return nil
	}

	return nil
}

// CreateSchemaMigrations sets up a table to track migrations. This is an idempotent
// operation.
func (m Migrator) CreateSchemaMigrations() error {
	return CreateSchemaMigrations(m.Connection, m.l)
}

// Status prints out the status of applied/pending migrations.
func (m Migrator) Status(out io.Writer) error {
	err := m.CreateSchemaMigrations()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', tabwriter.TabIndent)
	_, _ = fmt.Fprintln(w, "Version\tName\tStatus\t")
	for _, mf := range m.Migrations["up"] {
		exists, err := m.Connection.Where("version = ?", mf.Version).Exists(m.Connection.MigrationTableName())
		if err != nil {
			return errors.Wrapf(err, "problem with migration")
		}
		state := "Pending"
		if exists {
			state = "Applied"
		} else if len(mf.Version) > 14 {
			mtn := m.Connection.MigrationTableName()
			legacyVersion := mf.Version[:14]
			exists, err = m.Connection.Where("version = ?", legacyVersion).Exists(mtn)
			if err != nil {
				return errors.Wrapf(err, "problem checking for migration version %s", legacyVersion)
			}

			if exists {
				state = "Applied"
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t\n", mf.Version, mf.Name, state)
	}
	return w.Flush()
}

// DumpMigrationSchema will generate a file of the current database schema
// based on the value of Migrator.SchemaPath
func (m Migrator) DumpMigrationSchema() error {
	if m.SchemaPath == "" {
		return nil
	}
	c := m.Connection
	schema := filepath.Join(m.SchemaPath, "schema.sql")
	f, err := os.Create(schema)
	if err != nil {
		return err
	}
	err = c.Dialect.DumpSchema(f)
	if err != nil {
		os.RemoveAll(schema)
		return err
	}
	return nil
}

func (m Migrator) exec(fn func() error) error {
	now := time.Now()
	defer func() {
		err := m.DumpMigrationSchema()
		if err != nil {
			m.l.WithError(err).Warn("Migrator: unable to dump schema")
		}
	}()
	defer m.printTimer(now)

	err := m.CreateSchemaMigrations()
	if err != nil {
		return errors.Wrap(err, "migrator: problem creating schema migrations")
	}
	return fn()
}

func (m Migrator) printTimer(timerStart time.Time) {
	diff := time.Since(timerStart).Seconds()
	if diff > 60 {
		m.l.Debugf("%.4f minutes", diff/60)
	} else {
		m.l.Debugf("%.4f seconds", diff)
	}
}

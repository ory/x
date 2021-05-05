package popx

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cockroachdb/cockroach-go/v2/crdb"

	"github.com/ory/x/cmdx"

	"github.com/ory/x/tracing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"

	"github.com/gobuffalo/pop/v5"

	"github.com/ory/x/logrusx"

	"github.com/pkg/errors"
)

const (
	Pending = "Pending"
	Applied = "Applied"
)

var mrx = regexp.MustCompile(`^(\d+)_([^.]+)(\.[a-z0-9]+)?\.(up|down)\.(sql|fizz)$`)

// NewMigrator returns a new "blank" migrator. It is recommended
// to use something like MigrationBox or FileMigrator. A "blank"
// Migrator should only be used as the basis for a new type of
// migration system.
func NewMigrator(c *pop.Connection, l *logrusx.Logger, tracer *tracing.Tracer, perMigrationTimeout time.Duration) *Migrator {
	return &Migrator{
		Connection: c,
		l:          l,
		Migrations: map[string]Migrations{
			"up":   {},
			"down": {},
		},
		tracer:              tracer,
		PerMigrationTimeout: perMigrationTimeout,
	}
}

// Migrator forms the basis of all migrations systems.
// It does the actual heavy lifting of running migrations.
// When building a new migration system, you should embed this
// type into your migrator.
type Migrator struct {
	Connection          *pop.Connection
	SchemaPath          string
	Migrations          map[string]Migrations
	l                   *logrusx.Logger
	PerMigrationTimeout time.Duration
	tracer              *tracing.Tracer
}

func (m *Migrator) MigrationIsCompatible(dialect string, mi Migration) bool {
	if mi.DBType == "all" || mi.DBType == dialect {
		return true
	}
	return false
}

// Up runs pending "up" migrations and applies them to the database.
func (m *Migrator) Up(ctx context.Context) error {
	_, err := m.UpTo(ctx, 0)
	return err
}

// UpTo runs up to step "up" migrations and applies them to the database.
// If step <= 0 all pending migrations are run.
func (m *Migrator) UpTo(ctx context.Context, step int) (applied int, err error) {
	span, ctx := m.startSpan(ctx, MigrationUpOpName)
	defer span.Finish()
	span.LogFields(log.Int("up_to_step", step))

	c := m.Connection.WithContext(ctx)
	err = m.exec(ctx, func() error {
		mtn := m.migrationTableName(ctx, c)
		mfs := m.Migrations["up"].SortAndFilter(c.Dialect.Name())
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
					if err := m.isolatedTransaction(ctx, "init-migrate", func(tx *pop.Tx) error {
						// We do not want to remove the legacy migration version or subsequent migrations might be applied twice.
						//
						// Do not activate the following - it is just for reference.
						//
						// if _, err := tx.Store.Exec(fmt.Sprintf("DELETE FROM %s WHERE version = ?", mtn), legacyVersion); err != nil {
						//	return errors.Wrapf(err, "problem removing legacy version %s", mi.Version)
						// }

						// #nosec G201 - mtn is a system-wide const
						_, err := tx.Exec(tx.Rebind(fmt.Sprintf("INSERT INTO %s (version) VALUES (?)", mtn)), mi.Version)
						return errors.Wrapf(err, "problem inserting migration version %s", mi.Version)
					}); err != nil {
						return err
					}
					continue
				}
			}

			m.l.WithField("version", mi.Version).Debug("Migration has not yet been applied, running migration.")

			if err = m.isolatedTransaction(ctx, "up", func(tx *pop.Tx) error {
				if err := mi.Run(c, tx); err != nil {
					return err
				}

				// #nosec G201 - mtn is a system-wide const
				if _, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (version) VALUES ('%s')", mtn, mi.Version)); err != nil {
					return errors.Wrapf(err, "problem inserting migration version %s", mi.Version)
				}
				return nil
			}); err != nil {
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
func (m *Migrator) Down(ctx context.Context, step int) error {
	span, ctx := m.startSpan(ctx, MigrationDownOpName)
	defer span.Finish()

	c := m.Connection.WithContext(ctx)
	return m.exec(ctx, func() error {
		mtn := m.migrationTableName(ctx, c)
		count, err := c.Count(mtn)
		if err != nil {
			return errors.Wrap(err, "migration down: unable count existing migration")
		}
		mfs := m.Migrations["down"].SortAndFilter(c.Dialect.Name(), sort.Reverse)
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

			err = m.isolatedTransaction(ctx, "down", func(tx *pop.Tx) error {
				err := mi.Run(c, tx)
				if err != nil {
					return err
				}

				// #nosec G201 - mtn is a system-wide const
				if _, err = tx.Exec(tx.Rebind(fmt.Sprintf("DELETE FROM %s WHERE version = ?", mtn)), mi.Version); err != nil {
					return errors.Wrapf(err, "problem deleting migration version %s", mi.Version)
				}

				return nil
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
func (m *Migrator) Reset(ctx context.Context) error {
	err := m.Down(ctx, -1)
	if err != nil {
		return err
	}
	return m.Up(ctx)
}

func (m *Migrator) createTransactionalMigrationTable(ctx context.Context, c *pop.Connection, l *logrusx.Logger) error {
	mtn := m.migrationTableName(ctx, c)
	unprefixedMtn := m.migrationTableName(ctx, c)

	if err := m.execMigrationTransaction(ctx, c, []string{
		fmt.Sprintf(`CREATE TABLE %s (version VARCHAR (48) NOT NULL, version_self INT NOT NULL DEFAULT 0)`, mtn),
		fmt.Sprintf(`CREATE UNIQUE INDEX %s_version_idx ON %s (version)`, unprefixedMtn, mtn),
		fmt.Sprintf(`CREATE INDEX %s_version_self_idx ON %s (version_self)`, unprefixedMtn, mtn),
	}); err != nil {
		return err
	}

	l.WithField("migration_table", mtn).Debug("Transactional migration table created successfully.")

	return nil
}

func (m *Migrator) migrateToTransactionalMigrationTable(ctx context.Context, c *pop.Connection, l *logrusx.Logger) error {
	// This means the new pop migrator has also not yet been applied, do that now.
	mtn := m.migrationTableName(ctx, c)
	unprefixedMtn := m.migrationTableName(ctx, c)

	withOn := fmt.Sprintf(" ON %s", mtn)
	if c.Dialect.Name() != "mysql" {
		withOn = ""
	}

	interimTable := fmt.Sprintf("%s_transactional", mtn)
	workload := [][]string{
		{
			fmt.Sprintf(`DROP INDEX %s_version_idx%s`, unprefixedMtn, withOn),
			fmt.Sprintf(`CREATE TABLE %s (version VARCHAR (48) NOT NULL, version_self INT NOT NULL DEFAULT 0)`, interimTable),
			fmt.Sprintf(`CREATE UNIQUE INDEX %s_version_idx ON %s (version)`, unprefixedMtn, interimTable),
			fmt.Sprintf(`CREATE INDEX %s_version_self_idx ON %s (version_self)`, unprefixedMtn, interimTable),
			// #nosec G201 - mtn is a system-wide const
			fmt.Sprintf(`INSERT INTO %s (version) SELECT version FROM %s`, interimTable, mtn),
			fmt.Sprintf(`ALTER TABLE %s RENAME TO %s_pop_legacy`, mtn, mtn),
		},
		{
			fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, interimTable, mtn),
		},
	}

	if err := m.execMigrationTransaction(ctx, c, workload...); err != nil {
		return err
	}

	l.WithField("migration_table", mtn).Debug("Successfully migrated legacy schema_migration to new transactional schema_migration table.")

	return nil
}

func (m *Migrator) isolatedTransaction(ctx context.Context, direction string, fn func(tx *pop.Tx) error) error {
	span, ctx := m.startSpan(ctx, MigrationRunTransactionOpName)
	defer span.Finish()
	span.SetTag("migration_direction", direction)

	if m.PerMigrationTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.PerMigrationTimeout)
		defer cancel()
	}

	c := m.Connection.WithContext(ctx)
	tx, dberr := c.Store.TransactionContextOptions(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if dberr != nil {
		return dberr
	}

	err := fn(tx)
	if err != nil {
		dberr = tx.Rollback()
	} else {
		dberr = tx.Commit()
	}

	if dberr != nil {
		return errors.Wrapf(dberr, "error committing or rolling back transaction: %s", err)
	}

	return err
}

func (m *Migrator) execMigrationTransaction(ctx context.Context, c *pop.Connection, transactions ...[]string) error {
	for _, statements := range transactions {
		if err := m.isolatedTransaction(ctx, "init", func(tx *pop.Tx) error {
			for _, statement := range statements {
				if _, err := tx.ExecContext(ctx, statement); err != nil {
					return errors.Wrapf(err, "unable to execute statement: %s", statement)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// CreateSchemaMigrations sets up a table to track migrations. This is an idempotent
// operation.
func (m *Migrator) CreateSchemaMigrations(ctx context.Context) error {
	span, ctx := m.startSpan(ctx, MigrationInitOpName)
	defer span.Finish()

	c := m.Connection.WithContext(ctx)

	mtn := m.migrationTableName(ctx, c)
	m.l.WithField("migration_table", mtn).Debug("Checking if legacy migration table exists.")
	_, err := c.Store.Exec(fmt.Sprintf("select version from %s", mtn))
	if err != nil {
		m.l.WithError(err).WithField("migration_table", mtn).Debug("An error occurred while checking for the legacy migration table, maybe it does not exist yet? Trying to create.")
		// This means that the legacy pop migrator has not yet been applied
		return m.createTransactionalMigrationTable(ctx, c, m.l)
	}

	m.l.WithField("migration_table", mtn).Debug("A migration table exists, checking if it is a transactional migration table.")
	_, err = c.Store.Exec(fmt.Sprintf("select version, version_self from %s", mtn))
	if err != nil {
		m.l.WithError(err).WithField("migration_table", mtn).Debug("An error occurred while checking for the transactional migration table, maybe it does not exist yet? Trying to create.")
		return m.migrateToTransactionalMigrationTable(ctx, c, m.l)
	}

	m.l.WithField("migration_table", mtn).Debug("Migration tables exist and are up to date.")
	return nil
}

type MigrationStatus struct {
	State   string `json:"state"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

type MigrationStatuses []MigrationStatus

var _ cmdx.Table = (MigrationStatuses)(nil)

func (m MigrationStatuses) Header() []string {
	return []string{"Version", "Name", "Status"}
}

func (m MigrationStatuses) Table() [][]string {
	t := make([][]string, len(m))
	for i, s := range m {
		t[i] = []string{s.Version, s.Name, s.State}
	}
	return t
}

func (m MigrationStatuses) Interface() interface{} {
	return m
}

func (m MigrationStatuses) Len() int {
	return len(m)
}

func (m MigrationStatuses) IDs() []string {
	ids := make([]string, len(m))
	for i, s := range m {
		ids[i] = s.Version
	}
	return ids
}

// In the context of a cobra.Command, use cmdx.PrintTable instead.
func (m MigrationStatuses) Write(out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', tabwriter.TabIndent)
	_, _ = fmt.Fprintln(w, "Version\tName\tStatus\t")

	for _, mm := range m {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t\n", mm.Version, mm.Name, mm.State)
	}

	return w.Flush()
}

func (m MigrationStatuses) HasPending() bool {
	for _, mm := range m {
		if mm.State == Pending {
			return true
		}
	}
	return false
}

func (m *Migrator) migrationTableName(ctx context.Context, con *pop.Connection) string {
	return con.MigrationTableName()
}

func errIsTableNotFound(err error) bool {
	return strings.HasPrefix(err.Error(), "no such table:") || // sqlite
		strings.HasPrefix(err.Error(), "Error 1146:") || // MySQL
		strings.Contains(err.Error(), "SQLSTATE 42P01") // PostgreSQL / CockroachDB
}

// Status prints out the status of applied/pending migrations.
func (m *Migrator) Status(ctx context.Context) (MigrationStatuses, error) {
	span, ctx := m.startSpan(ctx, MigrationStatusOpName)
	defer span.Finish()

	con := m.Connection.WithContext(ctx)

	migrations := m.Migrations["up"].SortAndFilter(con.Dialect.Name())

	if len(migrations) == 0 {
		return nil, errors.Errorf("unable to find any migrations for dialect: %s", con.Dialect.Name())
	}

	statuses := make(MigrationStatuses, len(migrations))
	for k, mf := range migrations {
		statuses[k] = MigrationStatus{
			State:   Pending,
			Version: mf.Version,
			Name:    mf.Name,
		}

		exists, err := con.Where("version = ?", mf.Version).Exists(con.MigrationTableName())
		if err != nil {
			if errIsTableNotFound(err) {
				continue
			} else {
				return nil, errors.Wrapf(err, "problem with migration")
			}
		}

		if exists {
			statuses[k].State = Applied
		} else if len(mf.Version) > 14 {
			mtn := m.migrationTableName(ctx, con)
			legacyVersion := mf.Version[:14]
			exists, err = con.Where("version = ?", legacyVersion).Exists(mtn)
			if err != nil {
				return nil, errors.Wrapf(err, "problem checking for migration version %s", legacyVersion)
			}

			if exists {
				statuses[k].State = Applied
			}
		}
	}

	return statuses, nil
}

// DumpMigrationSchema will generate a file of the current database schema
// based on the value of Migrator.SchemaPath
func (m *Migrator) DumpMigrationSchema(ctx context.Context) error {
	if m.SchemaPath == "" {
		return nil
	}
	c := m.Connection.WithContext(ctx)
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

func (m *Migrator) wrapSpan(ctx context.Context, opName string, f func(ctx context.Context, span opentracing.Span) error) error {
	span, ctx := m.startSpan(ctx, opName)
	defer span.Finish()

	return f(ctx, span)
}

func (m *Migrator) startSpan(ctx context.Context, opName string) (opentracing.Span, context.Context) {
	tracer := opentracing.GlobalTracer()
	if m.tracer.IsLoaded() {
		tracer = m.tracer.Tracer()

	}

	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, opName)
	span.SetTag("component", "github.com/ory/x/popx")

	span.LogFields()
	return span, ctx
}

func (m *Migrator) exec(ctx context.Context, fn func() error) error {
	now := time.Now()
	defer func() {
		err := m.DumpMigrationSchema(ctx)
		if err != nil {
			m.l.WithError(err).Warn("Migrator: unable to dump schema")
		}
	}()
	defer m.printTimer(now)

	err := m.CreateSchemaMigrations(ctx)
	if err != nil {
		return errors.Wrap(err, "migrator: problem creating schema migrations")
	}

	if m.Connection.Dialect.Name() == "sqlite3" {
		if err := m.Connection.RawQuery("PRAGMA foreign_keys=OFF").Exec(); err != nil {
			return err
		}
	}

	if m.Connection.Dialect.Name() == "cockroach" {
		outer := fn
		fn = func() error {
			return crdb.Execute(outer)
		}
	}

	if err := fn(); err != nil {
		return err
	}

	if m.Connection.Dialect.Name() == "sqlite3" {
		if err := m.Connection.RawQuery("PRAGMA foreign_keys=ON").Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) printTimer(timerStart time.Time) {
	diff := time.Since(timerStart).Seconds()
	if diff > 60 {
		m.l.Debugf("%.4f minutes", diff/60)
	} else {
		m.l.Debugf("%.4f seconds", diff)
	}
}

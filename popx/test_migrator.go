package popx

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/x/logrusx"

	"github.com/gobuffalo/pop/v5"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// TestMigrator is a modified pop.FileMigrator
type TestMigrator struct {
	*Migrator
}

// Returns a new TestMigrator
// After running each migration it applies it's corresponding testData sql files.
// They are identified by having the same version (= number in the front of the filename).
// The filenames are expected to be of the format ([0-9]+).*(_testdata(\.[dbtype])?.sql
func NewTestMigrator(t *testing.T, c *pop.Connection, migrationPath, testDataPath string, l *logrusx.Logger) *TestMigrator {
	tm := TestMigrator{
		Migrator: NewMigrator(c, l, nil, time.Minute),
	}
	tm.SchemaPath = migrationPath
	testDataPath = strings.TrimSuffix(testDataPath, "/")

	runner := func(mf Migration, c *pop.Connection, tx *pop.Tx) error {
		b, err := ioutil.ReadFile(mf.Path)
		require.NoError(t, err)

		content, err := ParameterizedMigrationContent(nil)(mf, c, b, true)
		require.NoError(t, err)

		if len(strings.TrimSpace(content)) != 0 {
			_, err = tx.Exec(content)
			if err != nil {
				return errors.Wrapf(err, "error executing %s, sql: %s", mf.Path, content)
			}
		}

		t.Logf("Applied: %s", mf.Version)

		if mf.Direction != "up" {
			return nil
		}

		appliedVersion := mf.Version[:14]

		// find migration index
		if len(mf.Version) > 14 {
			upMigrations := tm.Migrations["up"].SortAndFilter(c.Dialect.Name())
			mgs := upMigrations

			require.False(t, len(mgs) == 0)

			var migrationIndex int = -1
			for k, m := range mgs {
				if m.Version == mf.Version {
					migrationIndex = k
					break
				}
			}

			require.NotEqual(t, -1, migrationIndex)

			if migrationIndex+1 > len(mgs)-1 {
				//
			} else {
				require.EqualValues(t, mf.Version, mgs[migrationIndex].Version)
				require.NotEqual(t, mf.Version, mgs[migrationIndex+1].Version)

				nextMigration := mgs[migrationIndex+1]
				if nextMigration.Version[:14] > appliedVersion {
					t.Logf("Executing transactional interim version %s (%s) because next is %s (%s)", mf.Version, appliedVersion, nextMigration.Version, nextMigration.Version[:14])
				} else if nextMigration.Version[:14] == appliedVersion {
					t.Logf("Skipping transactional interim version %s (%s) because next is %s (%s)", mf.Version, appliedVersion, nextMigration.Version, nextMigration.Version[:14])
					return nil
				} else {
					panic("asdf")
				}
			}
		}

		t.Logf("Adding migration test data %s (%s)", mf.Version, appliedVersion)

		// exec testdata
		var fileName string
		if fi, err := os.Stat(filepath.Join(testDataPath, appliedVersion+"_testdata."+c.Dialect.Name()+".sql")); err == nil && !fi.IsDir() {
			// found specific test data
			fileName = fi.Name()
		} else if fi, err := os.Stat(filepath.Join(testDataPath, appliedVersion+"_testdata.sql")); err == nil && !fi.IsDir() {
			// found generic test data
			fileName = fi.Name()
		} else {
			// found no test data
			t.Logf("Found no test data for migration %s %s", mf.Version, mf.DBType)
			return nil
		}

		// Workaround for https://github.com/cockroachdb/cockroach/issues/42643#issuecomment-611475836
		// This is not a problem as the test should fail anyway if there occurs any error
		// (either within a transaction or on it's own).
		if c.Dialect.Name() == "cockroach" && tx != nil {
			if err := tx.Commit(); err != nil {
				return errors.WithStack(err)
			}
			newTx, err := c.Store.TransactionContextOptions(context.Background(), &sql.TxOptions{
				Isolation: sql.LevelSerializable,
				ReadOnly:  false,
			})
			if err != nil {
				return errors.WithStack(err)
			}
			*tx = *newTx
		}

		data, err := ioutil.ReadFile(filepath.Join(testDataPath, fileName))
		if err != nil {
			return errors.WithStack(err)
		}

		if len(strings.TrimSpace(string(data))) == 0 {
			t.Logf("data is empty for: %s", fileName)
			return nil
		}

		// FIXME https://github.com/gobuffalo/pop/issues/567
		for _, statement := range strings.Split(string(data), ";\n") {
			t.Logf("Executing %s query from %s: %s", c.Dialect.Name(), fileName, statement)
			if strings.TrimSpace(statement) == "" {
				t.Logf("Skipping %s query from %s because empty: \"%s\"", c.Dialect.Name(), fileName, statement)
				continue
			}
			if _, err := tx.Exec(statement); err != nil {
				t.Logf("Unable to execute %s: %s", mf.Version, err)
				return errors.WithStack(err)
			}
		}

		return nil
	}

	if fi, err := os.Stat(migrationPath); err != nil || !fi.IsDir() {
		t.Fatalf("could not find directory %s", migrationPath)
		return nil
	}

	if fi, err := os.Stat(testDataPath); err != nil || !fi.IsDir() {
		t.Fatalf("could not find directory %s", testDataPath)
		return nil
	}

	require.NoError(t, filepath.Walk(migrationPath, func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			match, err := pop.ParseMigrationFilename(info.Name())
			if err != nil {
				return err
			}
			if match == nil {
				return nil
			}

			mf := Migration{
				Path:      p,
				Version:   match.Version,
				Name:      match.Name,
				DBType:    match.DBType,
				Direction: match.Direction,
				Type:      match.Type,
				Runner:    runner,
			}
			tm.Migrations[mf.Direction] = append(tm.Migrations[mf.Direction], mf)
		}
		return nil
	}))

	return &tm
}

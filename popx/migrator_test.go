// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package popx_test

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/gobuffalo/pop/v6"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/pkgerx"
	. "github.com/ory/x/popx"
	"github.com/ory/x/sqlcon/dockertest"
)

//go:embed stub/migrations/transactional/*.sql
var transactionalMigrations embed.FS

func TestMigratorUpgrading(t *testing.T) {
	litedb, err := ioutil.TempFile(os.TempDir(), "sqlite-*")
	require.NoError(t, err)
	require.NoError(t, litedb.Close())

	ctx := context.Background()

	sqlite, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "sqlite://file::memory:?_fk=true",
	})
	require.NoError(t, err)
	require.NoError(t, sqlite.Open())

	connections := map[string]*pop.Connection{
		"sqlite": sqlite,
	}

	if !testing.Short() {
		dockertest.Parallel([]func(){
			func() {
				connections["postgres"] = dockertest.ConnectToTestPostgreSQLPop(t)
			},
			func() {
				connections["mysql"] = dockertest.ConnectToTestMySQLPop(t)
			},
			func() {
				connections["cockroach"] = dockertest.ConnectToTestCockroachDBPop(t)
			},
		})
	}

	l := logrusx.New("", "", logrusx.ForceLevel(logrus.DebugLevel))

	for name, c := range connections {
		t.Run(fmt.Sprintf("database=%s", name), func(t *testing.T) {
			t.SkipNow()

			legacy, err := pkgerx.NewMigrationBox("/popx/stub/migrations/legacy", c, l)
			require.NoError(t, err)
			require.NoError(t, legacy.Up())

			var legacyStatusBuffer bytes.Buffer
			require.NoError(t, legacy.Status(&legacyStatusBuffer))

			legacyStatus := filterMySQL(t, name, legacyStatusBuffer.String())

			require.NotContains(t, legacyStatus, Pending)

			expected := legacy.DumpMigrationSchema()

			transactional, err := NewMigrationBox(transactionalMigrations, NewMigrator(c, l, nil, 0))
			require.NoError(t, err)

			var transactionalStatusBuffer bytes.Buffer
			statuses, err := transactional.Status(ctx)
			require.NoError(t, err)

			require.NoError(t, statuses.Write(&transactionalStatusBuffer))
			transactionalStatus := filterMySQL(t, name, transactionalStatusBuffer.String())
			require.NotContains(t, transactionalStatus, Pending)
			require.False(t, statuses.HasPending())

			require.NoError(t, transactional.Up(ctx))

			actual := transactional.DumpMigrationSchema(ctx)
			assert.EqualValues(t, expected, actual)

			// Re-set and re-try

			require.NoError(t, legacy.Down(-1))
			require.NoError(t, transactional.Up(ctx))
			actual = transactional.DumpMigrationSchema(ctx)
			assert.EqualValues(t, expected, actual)
		})
	}
}

func filterMySQL(t *testing.T, name string, status string) string {
	if name == "mysql" {
		return status
	}
	// These only run for mysql and are thus expected to be pending:
	//
	// 20191100000005   identities                                Pending
	// 20191100000009   verification                              Pending
	// 20200519101058   create_recovery_addresses                 Pending
	// 20200601101001   verification                              Pending

	pending := []string{"20191100000005", "20191100000009", "20200519101058", "20200601101001"}
	var lines []string
	for _, l := range strings.Split(status, "\n") {
		var skip bool
		for _, p := range pending {
			if strings.Contains(l, p) {
				t.Logf("Removing expected pending line: %s", l)
				skip = true
				break
			}
		}
		if !skip {
			lines = append(lines, l)
		}
	}

	return strings.Join(lines, "\n")
}

func TestMigratorUpgradingFromStart(t *testing.T) {
	litedb, err := ioutil.TempFile(os.TempDir(), "sqlite-*")
	require.NoError(t, err)
	require.NoError(t, litedb.Close())

	ctx := context.Background()

	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "sqlite://file::memory:?_fk=true",
	})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	l := logrusx.New("", "", logrusx.ForceLevel(logrus.DebugLevel))
	transactional, err := NewMigrationBox(transactionalMigrations, NewMigrator(c, l, nil, 0))
	require.NoError(t, err)
	status, err := transactional.Status(ctx)
	require.NoError(t, err)
	require.True(t, status.HasPending())

	require.NoError(t, transactional.Up(ctx))

	status, err = transactional.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.HasPending())

	// Are all the tables here?
	var rows []string
	require.NoError(t, c.Store.Select(&rows, "SELECT name FROM sqlite_master WHERE type='table'"))

	for _, expected := range []string{
		"schema_migration",
		"identities",
	} {
		require.Contains(t, rows, expected)
	}

	require.NoError(t, transactional.Down(ctx, -1))
}

func TestMigratorSanitizeMigrationTableName(t *testing.T) {
	litedb, err := ioutil.TempFile(os.TempDir(), "sqlite-*")
	require.NoError(t, err)
	require.NoError(t, litedb.Close())

	ctx := context.Background()

	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: `sqlite://file::memory:?_fk=true&migration_table_name=injection--`,
	})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	l := logrusx.New("", "", logrusx.ForceLevel(logrus.DebugLevel))
	transactional, err := NewMigrationBox(transactionalMigrations, NewMigrator(c, l, nil, 0))
	require.NoError(t, err)
	status, err := transactional.Status(ctx)
	require.NoError(t, err)
	require.True(t, status.HasPending())

	require.NoError(t, transactional.Up(ctx))

	status, err = transactional.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.HasPending())

	require.NoError(t, transactional.Down(ctx, -1))
}

package popx

import (
	"context"
	"embed"
	"github.com/gobuffalo/pop/v5"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/sqlcon/dockertest"
)

//go:embed stub/migrations/templating/*.sql
var migrations embed.FS

func TestMigrationBoxTemplating(t *testing.T) {
	templateVals := map[string]interface{}{
		"tableName": "test_table_name",
	}

	expectedMigration, err := migrations.ReadFile("stub/migrations/templating/0_sql_create_tablename_template.expected.sql")
	require.NoError(t, err)

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

	for n, c := range connections {
		t.Run("db="+n, func(t *testing.T) {
			mb, err := NewMigrationBox(migrations, NewMigrator(c, logrusx.New("", ""), nil, 0), WithTemplateValues(templateVals), WithMigrationContentMiddleware(func(content string, err error) (string, error) {
				require.NoError(t, err)
				assert.Equal(t, string(expectedMigration), content)

				return content, err
			}))
			require.NoError(t, err)

			err = mb.Up(context.Background())
			require.NoError(t, err)
		})
	}
}

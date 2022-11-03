// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package popx

import (
	"embed"
	"testing"

	"github.com/gobuffalo/pop/v6"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
)

//go:embed stub/migrations/templating/*.sql
var templatingMigrations embed.FS

func TestMigrationBoxTemplating(t *testing.T) {
	templateVals := map[string]interface{}{
		"tableName": "test_table_name",
	}

	expectedMigration, err := templatingMigrations.ReadFile("stub/migrations/templating/0_sql_create_tablename_template.expected.sql")
	require.NoError(t, err)

	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "sqlite://file::memory:?_fk=true",
	})
	require.NoError(t, err)
	require.NoError(t, c.Open())

	_, err = NewMigrationBox(
		templatingMigrations,
		NewMigrator(c, logrusx.New("", ""), nil, 0),
		WithTemplateValues(templateVals),
		WithMigrationContentMiddleware(func(content string, err error) (string, error) {
			require.NoError(t, err)
			assert.Equal(t, string(expectedMigration), content)
			return content, err
		}))
	require.NoError(t, err)
}

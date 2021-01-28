package pkgerx

import (
	"io/ioutil"
	"testing"

	"github.com/gobuffalo/pop/v5"
	"github.com/markbates/pkger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/sqlcon/dockertest"
)

// none of this works...
// var testData = pkger.Dir("github.com/ory/x:/pkgerx/testdata")
var testData = pkger.Dir("/pkgerx/testdata")

func TestMigrationBoxTemplating(t *testing.T) {
	for db, c := range map[string]*pop.Connection{
		"cockroach": dockertest.ConnectToTestCockroachDBPop(t),
		//"mysql": dockertest.ConnectToTestMySQLPop(t),
		//"postgres": dockertest.ConnectToTestPostgreSQLPop(t),
	} {
		mb, err := NewMigrationBox(testData, c, logrusx.New("", ""), WithTemplateValues(map[string]interface{}{
			"tableName": "test_table_name",
		}))
		require.NoError(t, err)
		require.NoError(t, mb.Up())

		dump := dockertest.DumpSchema(t, db)
		expected, err := ioutil.ReadFile("testdata/sql_create_tablename_template.sql")
		require.NoError(t, err)

		assert.Equal(t, string(expected), dump)
	}
}

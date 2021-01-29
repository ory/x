package pkgerx

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/gobuffalo/pop/v5"
	"github.com/markbates/pkger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/sqlcon/dockertest"
)

// none of this works...
var testData = pkger.Dir("github.com/ory/x:/pkgerx/testdata")

func TestMigrationBoxTemplating(t *testing.T) {
	t.Cleanup(dockertest.KillAllTestDatabases)
	templateVals := map[string]interface{}{
		"tableName": "test_table_name",
	}

	for db, c := range map[string]*pop.Connection{
		"cockroach": dockertest.ConnectToTestCockroachDBPop(t),
		"postgres":  dockertest.ConnectToTestPostgreSQLPop(t),
	} {
		t.Run("db="+db, func(t *testing.T) {
			mb, err := NewMigrationBox(testData, c, logrusx.New("", ""), WithTemplateValues(templateVals))
			require.NoError(t, err)
			require.NoError(t, mb.Up())

			dump := dockertest.DumpSchema(context.Background(), t, db)
			expectedDump, err := ioutil.ReadFile(filepath.Join("testdata", db+"_expected.sql"))
			require.NoError(t, err)

			assert.Equal(t, string(expectedDump), dump, "%#v", dump)
		})
	}
}

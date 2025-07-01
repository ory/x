// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package popx

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsMigrationEmpty(t *testing.T) {
	assert.True(t, isMigrationEmpty(""))
	assert.True(t, isMigrationEmpty("-- this is a comment"))
	assert.True(t, isMigrationEmpty(`

-- this is a comment

`))
	assert.False(t, isMigrationEmpty(`SELECT foo`))
	assert.False(t, isMigrationEmpty(`INSERT bar -- test`))
	assert.False(t, isMigrationEmpty(`
--test
INSERT bar -- test

`))
}

func TestMigrationSort(t *testing.T) {

	migrations := []Migration{
		{Version: "99", DBType: "mysql"},
		{Version: "98", DBType: "mysql"},
		{Version: "99", DBType: "sqlite"},
		{Version: "99", DBType: "all"},
		{Version: "97", DBType: "mysql"},
		{Version: "99", DBType: "postgresql"},
		{Version: "97", DBType: ""},
		{Version: "99", DBType: ""},
	}

	slices.SortFunc(migrations, CompareMigration)

	expected := []Migration{
		{Version: "97", DBType: ""},
		{Version: "97", DBType: "mysql"},
		{Version: "98", DBType: "mysql"},
		{Version: "99", DBType: ""},
		{Version: "99", DBType: "mysql"},
		{Version: "99", DBType: "postgresql"},
		{Version: "99", DBType: "sqlite"},
		{Version: "99", DBType: "all"},
	}
	assert.Equal(t, expected, migrations)
}

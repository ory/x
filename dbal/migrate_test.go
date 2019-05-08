package dbal

import (
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/stringslice"
)

func TestNewPackerMigrationSource(t *testing.T) {
	m, err := NewPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/b"}, false)
	require.NoError(t, err)
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/1.sql"), "%v", m.Box.List())
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/2.sql"), "%v", m.Box.List())
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/3.sql"), "%v", m.Box.List())

	m, err = NewPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/c"}, false)
	require.NoError(t, err)
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/1.sql"), "%v", m.Box.List())
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/2.sql"), "%v", m.Box.List())
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/3.sql"), "%v", m.Box.List())
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/4.sql"), "%v", m.Box.List())
}

func TestPackerOmitExtensionEnabled(t *testing.T) {
	m, err := NewPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/b"}, true)
	require.NoError(t, err)

	ms, err := m.FindMigrations()
	require.NoError(t, err)

	for _, mm := range ms {
		require.False(t, strings.Contains(mm.Id, ".sql"))
	}
}

func TestPackerOmitExtensionDisabled(t *testing.T) {
	m, err := NewPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/b"}, false)
	require.NoError(t, err)

	ms, err := m.FindMigrations()
	require.NoError(t, err)

	for _, mm := range ms {
		require.True(t, strings.Contains(mm.Id, ".sql"))
	}
}

func TestMigrationFileSort(t *testing.T) {
	m := migrationFiles{
		{Filename: "4.sql"},
		{Filename: "1.sql"},
		{Filename: "2.sql"},
		{Filename: "6.sql"},
	}
	sort.Sort(m)
	assert.EqualValues(t, migrationFiles{
		{Filename: "1.sql"},
		{Filename: "2.sql"},
		{Filename: "4.sql"},
		{Filename: "6.sql"},
	}, m)
}

func TestFindMatchingTestMigrations(t *testing.T) {
	m := map[string]*PackrMigrationSource{
		DriverMySQL:       NewMustPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a"}, false),
		DriverPostgreSQL:  NewMustPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/b"}, false),
		DriverCockroachDB: NewMustPackerMigrationSource(logrus.New(), AssetNames(), Asset, []string{"stub/a", "stub/c"}, false),
	}

	result := FindMatchingTestMigrations("stub/d/", m, AssetNames(), Asset)

	mysql := result[DriverMySQL]
	assert.True(t, stringslice.Has(mysql.Box.List(), "/migrations/sql/1_test.sql"), "%v", mysql.Box.List())
	assert.True(t, stringslice.Has(mysql.Box.List(), "/migrations/sql/3_test.sql"), "%v", mysql.Box.List())
	assert.True(t, len(mysql.Box.List()) == 2, "%v", len(mysql.Box.List()))

	postgres := result[DriverPostgreSQL]
	assert.True(t, stringslice.Has(postgres.Box.List(), "/migrations/sql/1_test.sql"), "%v", postgres.Box.List())
	assert.True(t, stringslice.Has(postgres.Box.List(), "/migrations/sql/2_test.sql"), "%v", postgres.Box.List())
	assert.True(t, stringslice.Has(postgres.Box.List(), "/migrations/sql/3_test.sql"), "%v", postgres.Box.List())
	assert.True(t, len(postgres.Box.List()) == 3, "%v", len(postgres.Box.List()))

	cockroach := result[DriverCockroachDB]
	assert.True(t, stringslice.Has(cockroach.Box.List(), "/migrations/sql/1_test.sql"), "%v", cockroach.Box.List())
	assert.True(t, stringslice.Has(cockroach.Box.List(), "/migrations/sql/2_test.sql"), "%v", cockroach.Box.List())
	assert.True(t, stringslice.Has(cockroach.Box.List(), "/migrations/sql/3_test.sql"), "%v", cockroach.Box.List())
	assert.True(t, stringslice.Has(cockroach.Box.List(), "/migrations/sql/4_test.sql"), "%v", cockroach.Box.List())
	assert.True(t, len(cockroach.Box.List()) == 4, "%v", len(cockroach.Box.List()))
}

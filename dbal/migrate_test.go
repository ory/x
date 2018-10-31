package dbal

import (
	"sort"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/go-convenience/stringslice"
)

func TestNewPackerMigrationSource(t *testing.T) {
	m, err := NewPackerMigrationSource(logrus.New(), []string{"./stub/a", "./stub/b"})
	require.NoError(t, err)
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/1.sql"))
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/2.sql"))
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/3.sql"))

	m, err = NewPackerMigrationSource(logrus.New(), []string{"./stub/a", "./stub/c"})
	require.NoError(t, err)
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/1.sql"))
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/2.sql"))
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/3.sql"))
	assert.True(t, stringslice.Has(m.Box.List(), "/migrations/sql/4.sql"))
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

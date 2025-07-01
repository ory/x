// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package popx

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

var migrations = Migrations{
	{
		Version: "1",
		DBType:  "all",
	},
	{
		Version: "1",
		DBType:  "postgres",
	},
	{
		Version: "2",
		DBType:  "cockroach",
	},
	{
		Version: "2",
		DBType:  "all",
	},
	{
		Version: "3",
		DBType:  "all",
	},
	{
		Version: "3",
		DBType:  "mysql",
	},
}

func TestFilterMigrations(t *testing.T) {
	t.Run("db=mysql", func(t *testing.T) {
		assert.Equal(t, Migrations{
			migrations[0],
			migrations[3],
			migrations[5],
		}, migrations.SortAndFilter("mysql"))
		assert.Equal(t, Migrations{
			migrations[5],
			migrations[3],
			migrations[0],
		}, migrations.SortAndFilter("mysql", sort.Reverse))
	})
}

func TestSortingMigrations(t *testing.T) {
	t.Run("case=enforces precedence for specific migrations", func(t *testing.T) {
		expectedOrder := Migrations{
			migrations[1],
			migrations[0],
			migrations[2],
			migrations[3],
			migrations[5],
			migrations[4],
		}

		sort.Sort(migrations)

		assert.Equal(t, expectedOrder, migrations)
	})
}

// From the docs:
// Less must describe a transitive ordering:
//   - if both Less(i, j) and Less(j, k) are true, then Less(i, k) must be true as well.
//   - if both Less(i, j) and Less(j, k) are false, then Less(i, k) must be false as well.
//
// Here:
// - i=0, j=1, k=2
// - i=2, j=1, k=0
// We only test the case of `a.Version == b.Version` because otherwise we just call the Go stdlib
// which is assumed to be correct.
func TestSortTransitiveOrdering(t *testing.T) {
	m := Migrations{
		{DBType: "b"}, {DBType: "c"}, {DBType: "all"},
	}

	assert.True(t, m.Less(0, 1))
	assert.True(t, m.Less(1, 2))
	assert.True(t, m.Less(0, 2))

	assert.False(t, m.Less(1, 0))
	assert.False(t, m.Less(2, 1))
	assert.False(t, m.Less(2, 0))
}

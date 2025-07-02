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

func isLesserThan(a, b Migration) bool {
	return -1 == CompareMigration(a, b)
}

// `slices.SortFunc` requires that `cmp` is a strict weak ordering: (https://en.wikipedia.org/wiki/Weak_ordering#Strict_weak_orderings.)
// - Irreflexivity: For all x ∈ S , it is not true that x < x .
// - Transitivity: For all x , y , z ∈ S , if x < y  and  y < z then x < z .
// - Asymmetry: For all x , y ∈ S , if x < y is true then y < x is false.
// - (there is a fourth rule which does not apply to us).
func TestSortStrictWeakOrdering(t *testing.T) {
	m := Migrations{
		{Version: "0", DBType: "b"}, {Version: "0", DBType: "c"}, {Version: "0", DBType: "all"}, {Version: "1", DBType: "d"},
	}

	// Irreflexivity.
	assert.False(t, isLesserThan(m[0], m[0]))
	assert.False(t, isLesserThan(m[1], m[1]))
	assert.False(t, isLesserThan(m[2], m[2]))
	assert.False(t, isLesserThan(m[3], m[3]))

	// Transitivity.
	assert.True(t, isLesserThan(m[0], m[1]))
	assert.True(t, isLesserThan(m[0], m[2]))
	assert.True(t, isLesserThan(m[0], m[3]))
	assert.True(t, isLesserThan(m[1], m[2]))
	assert.True(t, isLesserThan(m[1], m[3]))
	assert.True(t, isLesserThan(m[2], m[3]))

	// Asymmetry.
	assert.True(t, isLesserThan(m[0], m[1]))
	assert.False(t, isLesserThan(m[1], m[0]))

	assert.True(t, isLesserThan(m[0], m[2]))
	assert.False(t, isLesserThan(m[2], m[0]))

	assert.True(t, isLesserThan(m[0], m[3]))
	assert.False(t, isLesserThan(m[3], m[0]))

	assert.True(t, isLesserThan(m[1], m[2]))
	assert.False(t, isLesserThan(m[2], m[1]))

	assert.True(t, isLesserThan(m[1], m[3]))
	assert.False(t, isLesserThan(m[3], m[1]))

	assert.True(t, isLesserThan(m[2], m[3]))
	assert.False(t, isLesserThan(m[3], m[2]))
}

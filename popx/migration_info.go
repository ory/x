// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package popx

import (
	"fmt"
	"sort"

	"github.com/gobuffalo/pop/v6"
)

// Migration handles the data for a given database migration
type Migration struct {
	// Path to the migration (./migrations/123_create_widgets.up.sql)
	Path string
	// Version of the migration (123)
	Version string
	// Name of the migration (create_widgets)
	Name string
	// Direction of the migration (up)
	Direction string
	// Type of migration (sql)
	Type string
	// DB type (all|postgres|mysql...)
	DBType string
	// Runner function to run/execute the migration
	Runner func(Migration, *pop.Connection, *pop.Tx) error
}

// Run the migration. Returns an error if there is
// no mf.Runner defined.
func (mf Migration) Run(c *pop.Connection, tx *pop.Tx) error {
	if mf.Runner == nil {
		return fmt.Errorf("no runner defined for %s", mf.Path)
	}
	return mf.Runner(mf, c, tx)
}

// Migrations is a collection of Migration
type Migrations []Migration

func (mfs Migrations) Len() int {
	return len(mfs)
}

func (mfs Migrations) Less(i, j int) bool {
	if mfs[i].Version == mfs[j].Version {
		// force "all" to the back
		return mfs[i].DBType != "all"
	}
	return mfs[i].Version < mfs[j].Version
}

func (mfs Migrations) Swap(i, j int) {
	mfs[i], mfs[j] = mfs[j], mfs[i]
}

func (mfs Migrations) SortAndFilter(dialect string, modifiers ...func(sort.Interface) sort.Interface) Migrations {
	// We need to sort mfs in order to push the dbType=="all" migrations
	// to the back.
	m := append(Migrations{}, mfs...)
	sort.Sort(m)

	vsf := make(Migrations, 0)
	for k, v := range m {
		if v.DBType == "all" {
			// Add "all" only if we can not find a more specific migration for the dialect.
			var hasSpecific bool
			for kk, vv := range m {
				if v.Version == vv.Version && kk != k && vv.DBType == dialect {
					hasSpecific = true
					break
				}
			}

			if !hasSpecific {
				vsf = append(vsf, v)
			}
		} else if v.DBType == dialect {
			vsf = append(vsf, v)
		}
	}

	mod := sort.Interface(vsf)
	for _, m := range modifiers {
		mod = m(mod)
	}

	sort.Sort(mod)
	return vsf
}

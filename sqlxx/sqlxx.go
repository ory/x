// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package sqlxx

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx/reflectx"
)

// GetDBFieldNames extracts all database field names from a struct based on the `db` tags using sqlx.
// Fields without a `db` tag, with a `db:"-"` tag, or listed in the `exclude` parameter are omitted.
// Returns a slice of field names as strings.
func GetDBFieldNames(model interface{}, exclude ...string) []string {
	// Create a mapper that uses the "db" tag
	mapper := reflectx.NewMapper("db")

	// Get field names from the struct
	fields := mapper.TypeMap(reflectx.Deref(reflect.TypeOf(model))).Names

	// Extract just the field names
	fieldNames := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Field.Tag == "" || f.Path == "" || f.Name == "" || slices.Contains(exclude, f.Name) {
			continue
		}
		fieldNames = append(fieldNames, f.Name)
	}

	return fieldNames
}

// NamedInsertArguments returns columns and arguments for SQL INSERT statements based on a struct's tags. Does
// not work with nested structs or maps!
//
//	type st struct {
//		Foo string `db:"foo"`
//		Bar string `db:"bar,omitempty"`
//		Baz string `db:"-"`
//		Zab string
//	}
//	columns, arguments := NamedInsertArguments(new(st))
//	query := fmt.Sprintf("INSERT INTO foo (%s) VALUES (%s)", columns, arguments)
//	// INSERT INTO foo (foo, bar) VALUES (:foo, :bar)
func NamedInsertArguments(t any, exclude ...string) (columns string, arguments string) {
	keys := GetDBFieldNames(t, exclude...)
	return strings.Join(keys, ", "),
		":" + strings.Join(keys, ", :")
}

// NamedUpdateArguments returns columns and arguments for SQL UPDATE statements based on a struct's tags. Does
// not work with nested structs or maps!
//
//	type st struct {
//		Foo string `db:"foo"`
//		Bar string `db:"bar,omitempty"`
//		Baz string `db:"-"`
//		Zab string
//	}
//	query := fmt.Sprintf("UPDATE foo SET %s", NamedUpdateArguments(new(st)))
//	// UPDATE foo SET foo=:foo, bar=:bar
func NamedUpdateArguments(t any, exclude ...string) string {
	keys := GetDBFieldNames(t, exclude...)
	statements := make([]string, len(keys))

	for k, key := range keys {
		statements[k] = fmt.Sprintf("%s=:%s", key, key)
	}

	return strings.Join(statements, ", ")
}

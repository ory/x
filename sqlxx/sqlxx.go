// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package sqlxx

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"

	"github.com/ory/x/stringslice"
)

func keys(t interface{}, exclude []string) []string {
	s := structs.New(t)
	var keys []string
	for _, field := range s.Fields() {
		key := strings.Split(field.Tag("db"), ",")[0]
		if len(key) > 0 && key != "-" && !stringslice.Has(exclude, key) {
			keys = append(keys, key)
		}
	}

	return keys
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
func NamedInsertArguments(t interface{}, exclude ...string) (columns string, arguments string) {
	keys := keys(t, exclude)
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
func NamedUpdateArguments(t interface{}, exclude ...string) string {
	keys := keys(t, exclude)
	statements := make([]string, len(keys))

	for k, key := range keys {
		statements[k] = fmt.Sprintf("%s=:%s", key, key)
	}

	return strings.Join(statements, ", ")
}

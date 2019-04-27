package sqlxx

import (
	"fmt"
	"strings"

	"github.com/fatih/structs"
)

func keys(t interface{}) []string {
	s := structs.New(t)
	var keys []string
	for _, field := range s.Fields() {
		key := strings.Split(field.Tag("db"), ",")[0]
		if len(key) > 0 && key != "-" {
			keys = append(keys, key)
		}
	}

	return keys
}

// NamedInsertArguments returns columns and arguments for SQL INSERT statements based on a struct's tags. Does
// not work with nested structs or maps!
//
// 	type st struct {
// 		Foo string `db:"foo"`
// 		Bar string `db:"bar,omitempty"`
// 		Baz string `db:"-"`
// 		Zab string
// 	}
//	columns, arguments := NamedInsertArguments(new(st))
//	query := fmt.Sprintf("INSERT INTO foo (%s) VALUES (%s)", columns, arguments)
//	// INSERT INTO foo (foo, bar) VALUES (:foo, :bar)
func NamedInsertArguments(t interface{}) (columns string, arguments string) {
	keys := keys(t)
	return strings.Join(keys, ", "),
		":" + strings.Join(keys, ", :")
}

// NamedUpdateArguments returns columns and arguments for SQL UPDATE statements based on a struct's tags. Does
// not work with nested structs or maps!
//
// 	type st struct {
// 		Foo string `db:"foo"`
// 		Bar string `db:"bar,omitempty"`
// 		Baz string `db:"-"`
// 		Zab string
// 	}
//	query := fmt.Sprintf("UPDATE foo SET %s", NamedUpdateArguments(new(st)))
//	// UPDATE foo SET foo=:foo, bar=:bar
func NamedUpdateArguments(t interface{}) string {
	keys := keys(t)
	statements := make([]string, len(keys))

	for k, key := range keys {
		statements[k] = fmt.Sprintf("%s=:%s", key, key)
	}

	return strings.Join(statements, ", ")
}

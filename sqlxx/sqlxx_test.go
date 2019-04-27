package sqlxx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type st struct {
	Foo string `db:"foo"`
	Bar string `db:"bar,omitempty"`
	Baz string `db:"-"`
	Zab string
}

func TestNamedUpdateArguments(t *testing.T) {
	assert.Equal(t,
		"UPDATE foo SET foo=:foo, bar=:bar",
		fmt.Sprintf("UPDATE foo SET %s", NamedUpdateArguments(new(st))),
	)
}

func TestExpectNamedInsert(t *testing.T) {
	columns, arguments := NamedInsertArguments(new(st))
	assert.Equal(t,
		"INSERT INTO foo (foo, bar) VALUES (:foo, :bar)",
		fmt.Sprintf("INSERT INTO foo (%s) VALUES (%s)", columns, arguments),
	)
}

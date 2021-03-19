package dbal

import (
	"regexp"
)

const (
	SQLiteInMemory       = "sqlite://file::memory:?_fk=true"
	SQLiteSharedInMemory = "sqlite://file::memory:?_fk=true&cache=shared"
)

// SQLite can be written in different styles depending on the use case
// - just in memory
// - shared connection
// - shared but unique in the same process
// see: https://sqlite.org/inmemorydb.html
func IsMemorySQLite(dsn string) bool {
	if dsn == "memory" {
		return true
	}

	r := regexp.MustCompile(`(?P<a>sqlite://)(?P<b>(file:)?:?\w+)(?P<c>:\?_fk=true)(?P<d>&cache=shared)?(?P<e>&mode=memory)?(${a}${b}${c}${d}${e})?`)

	return r.MatchString(dsn)
}

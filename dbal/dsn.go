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
	r := regexp.MustCompile(`^(sqlite://file:(?:.+)\?((\w+=\w+)(&\w+=\w+)*)?(&?mode=memory)(&\w+=\w+)*)$|(?:sqlite://(file:)?:memory:(?:\?\w+=\w+)?(?:&\w+=\w+)*)|^(?:(?::memory:)|(?:memory))$`)
	return r.MatchString(dsn)
}

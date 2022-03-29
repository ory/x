package dbal

import (
	"fmt"
	"os"
	"regexp"
)

const (
	SQLiteInMemory       = "sqlite://file::memory:?_fk=true"
	SQLiteSharedInMemory = "sqlite://file::memory:?_fk=true&cache=shared"
)

var dsnRegex = regexp.MustCompile(`^(sqlite://file:(?:.+)\?((\w+=\w+)(&\w+=\w+)*)?(&?mode=memory)(&\w+=\w+)*)$|(?:sqlite://(file:)?:memory:(?:\?\w+=\w+)?(?:&\w+=\w+)*)|^(?:(?::memory:)|(?:memory))$`)

// SQLite can be written in different styles depending on the use case
// - just in memory
// - shared connection
// - shared but unique in the same process
// see: https://sqlite.org/inmemorydb.html
func IsMemorySQLite(dsn string) bool {
	return dsnRegex.MatchString(dsn)
}

// NewSharedUniqueInMemorySQLiteDatabase creates a new unique SQLite database
// which is shared amongst all callers and identified by an individual file name.
func NewSharedUniqueInMemorySQLiteDatabase() (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "unique-sqlite-db-*")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sqlite://file:%s/db.sqlite?_fk=true&mode=memory&cache=shared", dir), nil
}

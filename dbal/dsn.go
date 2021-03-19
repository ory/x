package dbal

import (
	"fmt"
	"github.com/ory/x/urlx"
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

	r := regexp.MustCompile(`sqlite://(file)?:?:\w+:`)

	url, err := urlx.Parse(dsn)

	if err != nil {
		return false
	}

	return r.MatchString(fmt.Sprintf("%s://%s", url.Scheme, url.Host))
}

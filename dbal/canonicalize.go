package dbal

import "github.com/ory/x/cmdx"

const (
	DriverMySQL      = "mysql"
	DriverPostgreSQL = "postgres"
	UnknownDriver    = "unknown"
)

func Canonicalize(database string) string {
	switch database {
	case "mysql":
		return DriverMySQL
	case "pgx", "pq", "postgres":
		return DriverPostgreSQL
	default:
		return UnknownDriver
	}
}

func MustCanonicalize(database string) string {
	d := Canonicalize(database)
	if d == UnknownDriver {
		cmdx.Fatalf("Unknown database driver: %s", database)
	}
	return d
}

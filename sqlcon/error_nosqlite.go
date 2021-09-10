//go:build !sqlite
// +build !sqlite

package sqlcon

// handleSqlite handles the error iff (if and only if) it is an sqlite error
func handleSqlite(_ error) error {
	return nil
}

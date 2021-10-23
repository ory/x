//go:build sqlite
// +build sqlite

package sqlcon

import (
	"strings"

	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

// handleSqlite handles the error iff (if and only if) it is an sqlite error
func handleSqlite(err error) error {
	if e := new(sqlite3.Error); errors.As(err, e) {
		switch e.ExtendedCode {
		case sqlite3.ErrConstraintUnique:
			fallthrough
		case sqlite3.ErrConstraintPrimaryKey:
			return errors.Wrap(ErrUniqueViolation, err.Error())
		}

		switch e.Code {
		case sqlite3.ErrError:
			if strings.Contains(err.Error(), "no such table") {
				return errors.Wrap(ErrNoSuchTable, err.Error())
			}
		}

		return errors.WithStack(err)
	}

	return nil
}

package sqlcon

import (
	"database/sql"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgconn"
	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/ory/herodot"

	"github.com/ory/x/errorsx"
)

var (
	// ErrUniqueViolation is returned when^a SQL INSERT / UPDATE command returns a conflict.
	ErrUniqueViolation = &herodot.DefaultError{
		CodeField:     http.StatusConflict,
		GRPCCodeField: codes.AlreadyExists,
		StatusField:   http.StatusText(http.StatusConflict),
		ErrorField:    "Unable to insert or update resource because a resource with that value exists already",
	}
	// ErrNoRows is returned when a SQL SELECT statement returns no rows.
	ErrNoRows = &herodot.DefaultError{
		CodeField:     http.StatusNotFound,
		GRPCCodeField: codes.NotFound,
		StatusField:   http.StatusText(http.StatusNotFound),
		ErrorField:    "Unable to locate the resource",
	}
	// ErrConcurrentUpdate is returned when the database is unable to serialize access due to a concurrent update.
	ErrConcurrentUpdate = &herodot.DefaultError{
		CodeField:     http.StatusBadRequest,
		GRPCCodeField: codes.Aborted,
		StatusField:   http.StatusText(http.StatusBadRequest),
		ErrorField:    "Unable to serialize access due to a concurrent update in another session",
	}
)

// HandleError returns the right sqlcon.Err* depending on the input error.
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errors.WithStack(ErrNoRows)
	}

	switch e := errorsx.Cause(err).(type) {
	case interface{ SQLState() string }:
		switch e.SQLState() {
		case "23505": // "unique_violation"
			return errors.Wrap(ErrUniqueViolation, err.Error())
		case "40001": // "serialization_failure"
			return errors.Wrap(ErrConcurrentUpdate, err.Error())
		}
	case *pq.Error:
		switch e.Code {
		case "23505": // "unique_violation"
			return errors.Wrap(ErrUniqueViolation, e.Error())
		case "40001": // "serialization_failure"
			return errors.Wrap(ErrConcurrentUpdate, e.Error())
		}
	case *mysql.MySQLError:
		switch e.Number {
		case 1062:
			return errors.Wrap(ErrUniqueViolation, err.Error())
		}
	case *pgconn.PgError:
		switch e.Code {
		case "23505": // "unique_violation"
			return errors.Wrap(ErrUniqueViolation, e.Error())
		case "40001": // "serialization_failure"
			return errors.Wrap(ErrConcurrentUpdate, e.Error())
		}
	}

	// Try other detections, for example for SQLite (we don't want to enforce CGO here!)
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return errors.Wrap(ErrUniqueViolation, err.Error())
	}

	return errors.WithStack(err)
}

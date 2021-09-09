package sqlcon

import (
	"database/sql"
	"net/http"

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
	ErrNoSuchTable = &herodot.DefaultError{
		CodeField:     http.StatusInternalServerError,
		GRPCCodeField: codes.Internal,
		StatusField:   http.StatusText(http.StatusInternalServerError),
		ErrorField:    "Unable to locate the table",
	}
)

func handlePostgres(err error, sqlState string) error {
	switch sqlState {
	case "23505": // "unique_violation"
		return errors.Wrap(ErrUniqueViolation, err.Error())
	case "40001": // "serialization_failure"
		return errors.Wrap(ErrConcurrentUpdate, err.Error())
	case "42P01": // "no such table"
		return errors.Wrap(ErrNoSuchTable, err.Error())
	}
	return errors.WithStack(err)
}

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
		return handlePostgres(err, e.SQLState())
	case *pq.Error:
		return handlePostgres(err, string(e.Code))
	case *pgconn.PgError:
		return handlePostgres(err, e.Code)
	case *mysql.MySQLError:
		switch e.Number {
		case 1062:
			return errors.Wrap(ErrUniqueViolation, err.Error())
		case 1146:
			return errors.Wrap(ErrNoSuchTable, e.Error())
		}
	}

	if err := handleSqlite(err); err != nil {
		return err
	}

	return errors.WithStack(err)
}

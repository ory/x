package watcherx

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// Import driver
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/sqlcon"
)

type row struct {
	key   string
	value string
}

func NewChangeFeedConnection(ctx context.Context, l *logrusx.Logger, dsn string) (*sqlx.DB, error) {
	if !strings.HasPrefix(dsn, "cockroach://") {
		return nil, errors.Errorf("DSN value must be prefixed with a cockroach URI schema")
	}

	_, _, _, _, cleanedDSN := sqlcon.ParseConnectionOptions(l, dsn)
	cleanedDSN = strings.Replace(dsn, "cockroach://", "postgres://", 1)
	l.WithField("component", "github.com/ory/x/watcherx.NewChangeFeedConnection").Info("Opening watcherx database connection.")
	cx, err := sqlx.Open("pgx", cleanedDSN)
	if err != nil {
		return nil, err
	}

	l.WithField("component", "github.com/ory/x/watcherx.NewChangeFeedConnection").Info("Connection to watcherx database is open.")

	cx.SetMaxIdleConns(1)
	cx.SetMaxOpenConns(1)
	cx.SetConnMaxLifetime(-1)
	cx.SetConnMaxIdleTime(-1)

	l.WithField("component", "github.com/ory/x/watcherx.NewChangeFeedConnection").Info("Trying to ping the watcherx database connection.")

	if err := cx.PingContext(ctx); err != nil {
		return nil, err
	}

	l.WithField("component", "github.com/ory/x/watcherx.NewChangeFeedConnection").Info("Enabling CHANGEFEED on watcherx database connection.")

	// Ensure CHANGEFEED is enabled
	_, err = cx.ExecContext(ctx, "SET CLUSTER SETTING kv.rangefeed.enabled = true")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	l.WithField("component", "github.com/ory/x/watcherx.NewChangeFeedConnection").Info("Initialization of CHANGEFEED is done.")

	return cx, nil
}

// WatchChangeFeed sends changed rows on the channel. To cancel the execution, cancel the context!
//
// Watcher.DispatchNow() does not have an effect in this method.
//
// This watcher is blocking to allow proper context cancellation and clean up.
func WatchChangeFeed(ctx context.Context, cx *sqlx.DB, tableName string, c EventChannel, cursor time.Time) (_ Watcher, err error) {
	var rows *sql.Rows
	if cursor.IsZero() {
		rows, err = cx.QueryContext(ctx, fmt.Sprintf("EXPERIMENTAL CHANGEFEED FOR %s", tableName))
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		var err error
		rows, err = cx.QueryContext(ctx, fmt.Sprintf("EXPERIMENTAL CHANGEFEED FOR %s WITH CURSOR = $1", tableName), fmt.Sprintf("%d", cursor.UnixNano()))
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	d := newDispatcher()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-d.trigger:
				d.done <- 0
			}
		}
	}()

	// basically run the watcher in a go routine which gets canceled either by the connection being closed
	// or by calling `"CANCEL QUERY"` below.
	done := make(chan struct{})
	go func() {
		defer func() {
			done <- struct{}{}
		}()

		for rows.Next() {
			var r row
			var table string

			if err := errors.WithStack(rows.Scan(&table, &r.key, &r.value)); err != nil {
				c <- &ErrorEvent{
					error: err,
				}
				continue
			}

			keys := gjson.Parse(r.key)
			eventSource := keys.Raw

			// For some reason this is an array - maybe because of composite primary keys?
			// See: https://www.cockroachlabs.com/docs/v20.2/changefeed-for.html
			if ka := keys.Array(); len(ka) > 0 {
				var ids []string
				for _, id := range ka {
					ids = append(ids, id.String())
				}

				eventSource = strings.Join(ids, "/")
			}

			after := gjson.Get(r.value, "after")
			if after.IsObject() {
				c <- &ChangeEvent{
					data:   []byte(after.Raw),
					source: source(eventSource),
				}
			} else {
				c <- &RemoveEvent{
					source: source(eventSource),
				}
			}
		}
	}()

	go func() {
		// naive attempt at context cancellation
		select {
		case <-ctx.Done():
		case <-done:
			close(done)
		}

		if err := rows.Close(); err != nil {
			c <- &ErrorEvent{
				error: err,
			}
			return
		}

		if err := cx.Close(); err != nil {
			c <- &ErrorEvent{
				error: err,
			}
			return
		}
		// end close
	}()

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return d, nil
}

package popx

import (
	"context"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/jmoiron/sqlx"

	"github.com/gobuffalo/pop/v6"
)

type transactionContextKey int

const transactionKey transactionContextKey = 0

func WithTransaction(ctx context.Context, tx *pop.Connection) context.Context {
	return context.WithValue(ctx, transactionKey, tx)
}

func Transaction(ctx context.Context, connection *pop.Connection, callback func(context.Context, *pop.Connection) error) error {
	c := ctx.Value(transactionKey)
	if c != nil {
		if conn, ok := c.(*pop.Connection); ok {
			return callback(ctx, conn.WithContext(ctx))
		}
	}

	if connection.Dialect.Name() == "cockroach" {
		return connection.WithContext(ctx).Dialect.Lock(func() error {
			transaction, err := connection.NewTransaction()
			if err != nil {
				return err
			}

			return crdb.ExecuteInTx(ctx, sqlxTxAdapter{transaction.TX.Tx}, func() error {
				return callback(WithTransaction(ctx, transaction), transaction)
			})
		})
	}

	return connection.WithContext(ctx).Transaction(func(tx *pop.Connection) error {
		return callback(WithTransaction(ctx, tx), tx)
	})
}

func GetConnection(ctx context.Context, connection *pop.Connection) *pop.Connection {
	c := ctx.Value(transactionKey)
	if c != nil {
		if conn, ok := c.(*pop.Connection); ok {
			return conn.WithContext(ctx)
		}
	}
	return connection.WithContext(ctx)
}

type sqlxTxAdapter struct {
	*sqlx.Tx
}

var _ crdb.Tx = sqlxTxAdapter{}

func (s sqlxTxAdapter) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := s.Tx.ExecContext(ctx, query, args...)
	return err
}

func (s sqlxTxAdapter) Commit(ctx context.Context) error {
	return s.Tx.Commit()
}

func (s sqlxTxAdapter) Rollback(ctx context.Context) error {
	return s.Tx.Rollback()
}

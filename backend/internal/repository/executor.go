package repository

import (
	"context"
	"database/sql"
)

// Executor is the smallest database contract needed by repositories and
// services, allowing them to work with both *sql.DB and *sql.Tx.
type Executor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

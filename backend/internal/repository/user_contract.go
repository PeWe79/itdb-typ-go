package repository

import (
	"context"
	"database/sql"
)

type UserRepository interface {
	Repository
	UsernameExists(context.Context, string, int64) (bool, error)
	FindUsername(context.Context, int64) (string, error)
	FindAdminID(context.Context) (int64, error)
	ReassignItems(context.Context, *sql.Tx, int64, int64) error
	DeleteUser(context.Context, *sql.Tx, int64) error
}

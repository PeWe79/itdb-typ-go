package service

import (
	"context"
	"database/sql"

	"itdb-backend/internal/repository"
)

// SQLService is the service-layer gateway for API database operations. It
// delegates storage details to repository.Store while preserving the existing
// database/sql contracts used by transactions and scanners.
type SQLService struct{ repo *repository.Store }

func NewSQLService(repo *repository.Store) *SQLService { return &SQLService{repo: repo} }
func (s *SQLService) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.repo.Query(query, args...)
}
func (s *SQLService) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.repo.QueryContext(ctx, query, args...)
}
func (s *SQLService) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.repo.QueryRow(query, args...)
}
func (s *SQLService) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.repo.QueryRowContext(ctx, query, args...)
}
func (s *SQLService) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.repo.Exec(query, args...)
}
func (s *SQLService) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.repo.ExecContext(ctx, query, args...)
}
func (s *SQLService) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.repo.BeginTx(ctx, opts)
}

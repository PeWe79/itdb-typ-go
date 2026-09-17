package repository

import (
	"context"
	"database/sql"
)

// DomainRepository is the SQLite implementation shared by the typed domain
// repository interfaces. It contains no HTTP or business-policy code.
type DomainRepository struct{ store *Store }

func NewDomainRepository(store *Store) *DomainRepository { return &DomainRepository{store: store} }
func (r *DomainRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return r.store.Query(query, args...)
}
func (r *DomainRepository) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.store.QueryContext(ctx, query, args...)
}
func (r *DomainRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	return r.store.QueryRow(query, args...)
}
func (r *DomainRepository) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return r.store.QueryRowContext(ctx, query, args...)
}
func (r *DomainRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	return r.store.Exec(query, args...)
}
func (r *DomainRepository) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return r.store.ExecContext(ctx, query, args...)
}
func (r *DomainRepository) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return r.store.BeginTx(ctx, opts)
}

func NewAuthRepository(s *Store) AuthRepository                     { return NewDomainRepository(s) }
func NewItemRepository(s *Store) ItemRepository                     { return NewDomainRepository(s) }
func NewSoftwareRepository(s *Store) SoftwareRepository             { return NewDomainRepository(s) }
func NewInvoiceRepository(s *Store) InvoiceRepository               { return NewDomainRepository(s) }
func NewContractRepository(s *Store) ContractRepository             { return NewDomainRepository(s) }
func NewFileRepository(s *Store) FileRepository                     { return NewDomainRepository(s) }
func NewAgentRepository(s *Store) AgentRepository                   { return NewDomainRepository(s) }
func NewUserRepository(s *Store) UserRepository                     { return NewDomainRepository(s) }
func NewInfrastructureRepository(s *Store) InfrastructureRepository { return NewDomainRepository(s) }
func NewDictionaryRepository(s *Store) DictionaryRepository         { return NewDomainRepository(s) }
func NewTagRepository(s *Store) TagRepository                       { return NewDomainRepository(s) }
func NewSystemRepository(s *Store) SystemRepository                 { return NewDomainRepository(s) }
func NewToolRepository(s *Store) ToolRepository                     { return NewDomainRepository(s) }

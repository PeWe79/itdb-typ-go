package service

import (
	"context"
	"database/sql"

	"itdb-backend/internal/repository"
)

// RepositoryService is the common service contract exposed to HTTP adapters.
type RepositoryService interface {
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type AuthService interface{ RepositoryService }
type ItemService interface{ RepositoryService }
type SoftwareService interface{ RepositoryService }
type InvoiceService interface{ RepositoryService }
type ContractService interface{ RepositoryService }
type FileService interface{ RepositoryService }
type AgentService interface{ RepositoryService }
type UserService interface{ RepositoryService }
type InfrastructureService interface{ RepositoryService }
type DictionaryService interface{ RepositoryService }
type TagService interface{ RepositoryService }
type SystemService interface{ RepositoryService }
type ToolService interface{ RepositoryService }

type domainService struct{ repo repository.Repository }

func newDomainService(repo repository.Repository) *domainService { return &domainService{repo: repo} }
func (s *domainService) Query(q string, a ...interface{}) (*sql.Rows, error) {
	return s.repo.Query(q, a...)
}
func (s *domainService) QueryContext(c context.Context, q string, a ...interface{}) (*sql.Rows, error) {
	return s.repo.QueryContext(c, q, a...)
}
func (s *domainService) QueryRow(q string, a ...interface{}) *sql.Row {
	return s.repo.QueryRow(q, a...)
}
func (s *domainService) QueryRowContext(c context.Context, q string, a ...interface{}) *sql.Row {
	return s.repo.QueryRowContext(c, q, a...)
}
func (s *domainService) Exec(q string, a ...interface{}) (sql.Result, error) {
	return s.repo.Exec(q, a...)
}
func (s *domainService) ExecContext(c context.Context, q string, a ...interface{}) (sql.Result, error) {
	return s.repo.ExecContext(c, q, a...)
}
func (s *domainService) BeginTx(c context.Context, o *sql.TxOptions) (*sql.Tx, error) {
	return s.repo.BeginTx(c, o)
}

type DomainServices struct {
	Auth           AuthService
	Items          ItemService
	Software       SoftwareService
	Invoices       InvoiceService
	Contracts      ContractService
	Files          FileService
	Agents         AgentService
	Users          UserService
	Infrastructure InfrastructureService
	Dictionaries   DictionaryService
	Tags           TagService
	System         SystemService
	Tools          ToolService
}

func NewDomainServices(store *repository.Store) *DomainServices {
	return &DomainServices{
		Auth: newDomainService(repository.NewAuthRepository(store)), Items: newDomainService(repository.NewItemRepository(store)), Software: newDomainService(repository.NewSoftwareRepository(store)),
		Invoices: newDomainService(repository.NewInvoiceRepository(store)), Contracts: newDomainService(repository.NewContractRepository(store)), Files: newDomainService(repository.NewFileRepository(store)),
		Agents: newDomainService(repository.NewAgentRepository(store)), Users: newDomainService(repository.NewUserRepository(store)), Infrastructure: newDomainService(repository.NewInfrastructureRepository(store)),
		Dictionaries: newDomainService(repository.NewDictionaryRepository(store)), Tags: newDomainService(repository.NewTagRepository(store)), System: newDomainService(repository.NewSystemRepository(store)), Tools: newDomainService(repository.NewToolRepository(store)),
	}
}

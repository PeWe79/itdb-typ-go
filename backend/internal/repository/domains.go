package repository

import (
	"context"
	"database/sql"
	"time"
)

// Each interface represents the persistence boundary of one business domain.
// They intentionally expose only SQL operations during this compatibility
// migration; domain-specific query methods can be added without changing HTTP.
type AuthRepository interface {
	Repository
	FindAuthUser(context.Context, string) (AuthUserRecord, error)
	FindAuthUserByWecom(context.Context, string) (AuthUserRecord, error)
	UserDescription(context.Context, int64) (string, error)
	UpdateAuthPassword(context.Context, int64, string) error
	CreateSession(context.Context, SessionRecord) error
	FindSession(context.Context, string) (SessionRecord, error)
	DeleteSession(context.Context, string) error
	DeleteExpiredSessions(context.Context, time.Time) error
}
type ItemRepository interface {
	Repository
	ListItems(context.Context, string, int64, int64) (*sql.Rows, error)
	GetItem(context.Context, int64) (*sql.Rows, error)
	ListItemActions(context.Context, int64) (*sql.Rows, error)
	InsertItem(context.Context, *sql.Tx, ...interface{}) (sql.Result, error)
	UpdateItem(context.Context, *sql.Tx, int64, ...interface{}) (sql.Result, error)
	ItemUserID(context.Context, *sql.Tx, int64) (sql.NullInt64, error)
	ItemFileIDs(context.Context, *sql.Tx, int64) ([]int64, error)
	DeleteItemReferences(context.Context, *sql.Tx, int64) error
	RecordItemAction(context.Context, int64, string, string) error
}
type SoftwareRepository interface {
	Repository
	ListSoftware(context.Context, string, int64, int64) (*sql.Rows, error)
	GetSoftware(context.Context, int64) (*sql.Rows, error)
	InsertSoftware(context.Context, *sql.Tx, ...interface{}) (sql.Result, error)
	UpdateSoftware(context.Context, *sql.Tx, int64, ...interface{}) (sql.Result, error)
	SoftwareFileIDs(context.Context, *sql.Tx, int64) ([]int64, error)
	DeleteSoftwareReferences(context.Context, *sql.Tx, int64) error
}
type InvoiceRepository interface {
	Repository
	ListInvoices(context.Context, string, int64, int64) (*sql.Rows, error)
	GetInvoice(context.Context, int64) (*sql.Rows, error)
	InsertInvoice(context.Context, *sql.Tx, ...interface{}) (sql.Result, error)
	UpdateInvoice(context.Context, *sql.Tx, int64, ...interface{}) (sql.Result, error)
	InvoiceFileIDs(context.Context, *sql.Tx, int64) ([]int64, error)
	DeleteInvoiceReferences(context.Context, *sql.Tx, int64) error
}
type ContractRepository interface {
	Repository
	ListContracts(context.Context, string) (*sql.Rows, error)
	GetContract(context.Context, int64) (*sql.Rows, error)
	InsertContract(context.Context, *sql.Tx, ...interface{}) (sql.Result, error)
	UpdateContract(context.Context, *sql.Tx, int64, ...interface{}) (sql.Result, error)
	ContractFileIDs(context.Context, *sql.Tx, int64) ([]int64, error)
	DeleteContractReferences(context.Context, *sql.Tx, int64) error
	ListContractEvents(context.Context, int64) (*sql.Rows, error)
	CreateContractEvent(context.Context, int64, int64, int64, int64, string) (sql.Result, error)
	UpdateContractEvent(context.Context, int64, int64, int64, int64, string) (sql.Result, error)
	DeleteContractEvent(context.Context, int64) (sql.Result, error)
}
type FileRepository interface{ Repository }
type AgentRepository interface {
	Repository
	ListAgents(context.Context, string) (*sql.Rows, error)
	GetAgent(context.Context, int64) (*sql.Rows, error)
	CreateAgent(context.Context, int64, string, string, string, string) (sql.Result, error)
	UpdateAgent(context.Context, int64, int64, string, string, string, string) (sql.Result, error)
	DeleteAgentReferences(context.Context, *sql.Tx, int64) error
}
type InfrastructureRepository interface {
	ListLocations(context.Context, string) (*sql.Rows, error)
	GetLocation(context.Context, int64) (*sql.Rows, error)
	ListAreas(context.Context, int64) (*sql.Rows, error)
	AreaRackCount(context.Context, int64) (int64, error)
	AreaItemCount(context.Context, int64) (int64, error)
	LocationItemCount(context.Context, int64) (int64, error)
	LocationRackCount(context.Context, int64) (int64, error)
	CreateLocation(context.Context, string, string, string) (sql.Result, error)
	UpdateLocation(context.Context, *sql.Tx, int64, string, string, *string) (sql.Result, error)
	LocationFloorplan(context.Context, int64) (string, string, error)
	DeleteLocation(context.Context, *sql.Tx, int64) error
	Repository
	ListRacks(context.Context, string) (*sql.Rows, error)
	GetRack(context.Context, int64) (*sql.Rows, error)
	InsertRack(context.Context, ...interface{}) (sql.Result, error)
	UpdateRack(context.Context, *sql.Tx, int64, ...interface{}) (sql.Result, error)
	RackItemCount(context.Context, int64) (int64, error)
	UpdateRackItems(context.Context, *sql.Tx, int64, int64, interface{}) error
	DeleteRack(context.Context, *sql.Tx, int64) error
}
type DictionaryRepository interface{ Repository }
type TagRepository interface{ Repository }
type SystemRepository interface{ Repository }
type ToolRepository interface {
	ListLabelItems(context.Context, string, string, int64, int64) (*sql.Rows, error)
	PreviewLabelItems(context.Context, []int64) (*sql.Rows, error)
	ListLabelPresets(context.Context) (*sql.Rows, error)
	CreateLabelPreset(context.Context, ...interface{}) (sql.Result, error)
	FindLabelPresetIDByName(context.Context, string) (int64, error)
	UpdateLabelPreset(context.Context, int64, ...interface{}) error
	DeleteLabelPreset(context.Context, int64) (sql.Result, error)
	Repository
	BrowseRows(context.Context, string) (*sql.Rows, string, bool, error)
}

// Repository is the common persistence contract used by domain services.
type Repository interface {
	Executor
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

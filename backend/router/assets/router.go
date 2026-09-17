package assets

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"

	"itdb-backend/config"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
)

type Router struct {
	db                       *sql.DB
	sql                      *service.SQLService
	queries                  *repository.SQL
	domains                  *service.DomainServices
	audit                    *service.AuditService
	files                    *service.FileStorageService
	relations                *service.RelationService
	tagWorkflow              *service.TagWorkflow
	agentWorkflow            *service.AgentWorkflow
	itemWorkflow             *service.ItemWorkflow
	itemActionWorkflow       *service.ItemActionWorkflow
	itemMutationWorkflow     *service.ItemMutationWorkflow
	itemTagWorkflow          *service.ItemTagWorkflow
	softwareWorkflow         *service.SoftwareWorkflow
	softwareMutationWorkflow *service.SoftwareMutationWorkflow
	softwareTagWorkflow      *service.SoftwareTagWorkflow
	invoiceWorkflow          *service.InvoiceWorkflow
	contractWorkflow         *service.ContractWorkflow
	contractEventWorkflow    *service.ContractEventWorkflow
	rackWorkflow             *service.RackWorkflow
	locationWorkflow         *service.LocationWorkflow
	detailRelations          *service.DetailRelationsWorkflow
	browseWorkflow           *service.BrowseWorkflow
	reportWorkflow           *service.ReportWorkflow
	labelWorkflow            *service.LabelWorkflow
	labelPresetWorkflow      *service.LabelPresetWorkflow
	currentUserPermissions   func(*http.Request) []string
	cfg                      config.Config
}

// Deps 聚合资产域路由依赖，由根路由统一注入
type Deps struct {
	DB                       *sql.DB
	SQL                      *service.SQLService
	Queries                  *repository.SQL
	Domains                  *service.DomainServices
	Audit                    *service.AuditService
	Files                    *service.FileStorageService
	Relations                *service.RelationService
	TagWorkflow              *service.TagWorkflow
	AgentWorkflow            *service.AgentWorkflow
	ItemWorkflow             *service.ItemWorkflow
	ItemActionWorkflow       *service.ItemActionWorkflow
	ItemMutationWorkflow     *service.ItemMutationWorkflow
	ItemTagWorkflow          *service.ItemTagWorkflow
	SoftwareWorkflow         *service.SoftwareWorkflow
	SoftwareMutationWorkflow *service.SoftwareMutationWorkflow
	SoftwareTagWorkflow      *service.SoftwareTagWorkflow
	InvoiceWorkflow          *service.InvoiceWorkflow
	ContractWorkflow         *service.ContractWorkflow
	ContractEventWorkflow    *service.ContractEventWorkflow
	RackWorkflow             *service.RackWorkflow
	LocationWorkflow         *service.LocationWorkflow
	DetailRelations          *service.DetailRelationsWorkflow
	BrowseWorkflow           *service.BrowseWorkflow
	ReportWorkflow           *service.ReportWorkflow
	LabelWorkflow            *service.LabelWorkflow
	LabelPresetWorkflow      *service.LabelPresetWorkflow
	CurrentUserPermissions   func(*http.Request) []string
	Cfg                      config.Config
}

func New(d Deps) *Router {
	return &Router{
		db:                       d.DB,
		sql:                      d.SQL,
		queries:                  d.Queries,
		domains:                  d.Domains,
		audit:                    d.Audit,
		files:                    d.Files,
		relations:                d.Relations,
		tagWorkflow:              d.TagWorkflow,
		agentWorkflow:            d.AgentWorkflow,
		itemWorkflow:             d.ItemWorkflow,
		itemActionWorkflow:       d.ItemActionWorkflow,
		itemMutationWorkflow:     d.ItemMutationWorkflow,
		itemTagWorkflow:          d.ItemTagWorkflow,
		softwareWorkflow:         d.SoftwareWorkflow,
		softwareMutationWorkflow: d.SoftwareMutationWorkflow,
		softwareTagWorkflow:      d.SoftwareTagWorkflow,
		invoiceWorkflow:          d.InvoiceWorkflow,
		contractWorkflow:         d.ContractWorkflow,
		contractEventWorkflow:    d.ContractEventWorkflow,
		rackWorkflow:             d.RackWorkflow,
		locationWorkflow:         d.LocationWorkflow,
		detailRelations:          d.DetailRelations,
		browseWorkflow:           d.BrowseWorkflow,
		reportWorkflow:           d.ReportWorkflow,
		labelWorkflow:            d.LabelWorkflow,
		labelPresetWorkflow:      d.LabelPresetWorkflow,
		currentUserPermissions:   d.CurrentUserPermissions,
		cfg:                      d.Cfg,
	}
}

// Register 挂载资产域路由，并按资源粒度绑定查看/管理权限
func (a *Router) Register(r chi.Router, requirePermission func(string) func(http.Handler) http.Handler, currentUserPermissions func(*http.Request) []string) {
	a.currentUserPermissions = currentUserPermissions
	read := func(resource string) func(http.Handler) http.Handler {
		return requirePermission(assetPermission(resource, "read"))
	}
	manage := func(resource string) func(http.Handler) http.Handler {
		return requirePermission(assetPermission(resource, "manage"))
	}

	r.With(read("items")).Get("/api/items", a.handleListItems)
	r.With(read("items")).Get("/api/items/{id}", a.handleGetItem)
	r.With(read("items")).Get("/api/items/{id}/actions", a.handleListItemActions)
	r.With(manage("items")).Post("/api/items", a.handleCreateItem)
	r.With(manage("items")).Put("/api/items/{id}", a.handleUpdateItem)
	r.With(manage("items")).Delete("/api/items/{id}", a.handleDeleteItem)
	r.With(manage("items")).Post("/api/items/{id}/tags", a.handleMutateItemTag)

	r.With(read("software")).Get("/api/software", a.handleListSoftware)
	r.With(read("software")).Get("/api/software/{id}", a.handleGetSoftware)
	r.With(manage("software")).Post("/api/software", a.handleCreateSoftware)
	r.With(manage("software")).Put("/api/software/{id}", a.handleUpdateSoftware)
	r.With(manage("software")).Delete("/api/software/{id}", a.handleDeleteSoftware)
	r.With(manage("software")).Post("/api/software/{id}/tags", a.handleMutateSoftwareTag)

	r.With(read("invoices")).Get("/api/invoices", a.handleListInvoices)
	r.With(read("invoices")).Get("/api/invoices/{id}", a.handleGetInvoice)
	r.With(manage("invoices")).Post("/api/invoices", a.handleCreateInvoice)
	r.With(manage("invoices")).Put("/api/invoices/{id}", a.handleUpdateInvoice)
	r.With(manage("invoices")).Delete("/api/invoices/{id}", a.handleDeleteInvoice)

	r.With(read("contracts")).Get("/api/contracts", a.handleListContracts)
	r.With(read("contracts")).Get("/api/contracts/{id}", a.handleGetContract)
	r.With(read("contracts")).Get("/api/contracts/next-event-id", a.handleNextContractEventID)
	r.With(read("contracts")).Get("/api/contracts/{id}/events", a.handleListContractEvents)
	r.With(manage("contracts")).Post("/api/contracts", a.handleCreateContract)
	r.With(manage("contracts")).Put("/api/contracts/{id}", a.handleUpdateContract)
	r.With(manage("contracts")).Delete("/api/contracts/{id}", a.handleDeleteContract)
	r.With(manage("contracts")).Post("/api/contracts/{id}/events", a.handleCreateContractEvent)
	r.With(manage("contracts")).Put("/api/contracts/{id}/events/{eventId}", a.handleUpdateContractEvent)
	r.With(manage("contracts")).Delete("/api/contracts/{id}/events/{eventId}", a.handleDeleteContractEvent)

	r.With(read("files")).Get("/api/files", a.handleListFiles)
	r.With(read("files")).Get("/api/files/{id}", a.handleGetFile)
	r.Get("/api/files/{id}/download", a.handleDownloadFile)
	r.With(manage("files")).Post("/api/files", a.handleCreateFile)
	r.With(manage("files")).Put("/api/files/{id}", a.handleUpdateFile)
	r.With(manage("files")).Delete("/api/files/{id}", a.handleDeleteFile)

	r.With(read("agents")).Get("/api/agents", a.handleListAgents)
	r.With(read("agents")).Get("/api/agents/{id}", a.handleGetAgent)
	r.With(manage("agents")).Post("/api/agents", a.handleCreateAgent)
	r.With(manage("agents")).Put("/api/agents/{id}", a.handleUpdateAgent)
	r.With(manage("agents")).Delete("/api/agents/{id}", a.handleDeleteAgent)

	r.With(read("locations")).Get("/api/locations", a.handleListLocations)
	r.With(read("locations")).Get("/api/locations/next-area-id", a.handleNextLocAreaID)
	r.With(read("locations")).Get("/api/locations/{id}", a.handleGetLocation)
	r.With(read("locations")).Get("/api/locations/{id}/floorplan", a.handleDownloadLocationFloorplan)
	r.With(read("locations")).Get("/api/locations/{id}/areas", a.handleListLocAreas)
	r.With(manage("locations")).Post("/api/locations", a.handleCreateLocation)
	r.With(manage("locations")).Put("/api/locations/{id}", a.handleUpdateLocation)
	r.With(manage("locations")).Delete("/api/locations/{id}", a.handleDeleteLocation)
	r.With(manage("locations")).Post("/api/locations/{id}/areas", a.handleCreateLocArea)
	r.With(manage("locations")).Put("/api/locations/{id}/areas/{areaId}", a.handleUpdateLocArea)
	r.With(manage("locations")).Delete("/api/locations/{id}/areas/{areaId}", a.handleDeleteLocArea)

	r.With(read("racks")).Get("/api/racks", a.handleListRacks)
	r.With(read("racks")).Get("/api/racks/{id}", a.handleGetRack)
	r.With(manage("racks")).Post("/api/racks", a.handleCreateRack)
	r.With(manage("racks")).Put("/api/racks/{id}", a.handleUpdateRack)
	r.With(manage("racks")).Delete("/api/racks/{id}", a.handleDeleteRack)

	r.Get("/api/dictionaries", a.handleListDictionaries)
	r.Post("/api/dictionaries/{name}", a.handleCreateDictionaryRow)
	r.Put("/api/dictionaries/{name}/{id}", a.handleUpdateDictionaryRow)
	r.Delete("/api/dictionaries/{name}/{id}", a.handleDeleteDictionaryRow)
	r.Post("/api/history/events", a.handleTrackAuditEvent)
	r.With(requirePermission(dictionaryPermission("tags", "manage"))).Get("/api/tags/next-id", a.handleNextTagID)
	r.With(requirePermission(dictionaryPermission("tags", "read"))).Get("/api/tags/{id}/items", a.handleListTagItems)
	r.With(requirePermission(dictionaryPermission("tags", "read"))).Get("/api/tags/{id}/software", a.handleListTagSoftware)

	r.With(requirePermission("reports.read")).Get("/api/reports", a.handleListReports)
	r.With(requirePermission("reports.read")).Get("/api/reports/{name}", a.handleRunReport)

	r.With(requirePermission("browse.read")).Get("/api/browse/tree", a.handleBrowseTree)

	r.With(requirePermission("labels.preview")).Get("/api/labels/items", a.handleListLabelItems)
	r.With(requirePermission("labels.preview")).Get("/api/labels/presets", a.handleListLabelPresets)
	r.With(requirePermission("labels.preview")).Post("/api/labels/preview", a.handlePreviewLabels)
	r.With(requirePermission("labels.manage")).Post("/api/labels/presets", a.handleCreateLabelPreset)
	r.With(requirePermission("labels.manage")).Delete("/api/labels/presets/{id}", a.handleDeleteLabelPreset)
}

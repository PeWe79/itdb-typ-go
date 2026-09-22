package router

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"itdb-backend/router/common"
	"itdb-backend/router/settings"
	"itdb-backend/router/system"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"itdb-backend/config"
	_ "itdb-backend/docs"
	"itdb-backend/internal/buildinfo"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/service"
	"itdb-backend/router/assets"
	"itdb-backend/router/auth"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

type App struct {
	db                       *sql.DB
	sql                      *service.SQLService
	domains                  *service.DomainServices
	queries                  *repository.SQL
	audit                    *service.AuditService
	files                    *service.FileStorageService
	tagWorkflow              *service.TagWorkflow
	userWorkflow             *service.UserWorkflow
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
	authWorkflow             *service.AuthWorkflow
	detailRelations          *service.DetailRelationsWorkflow
	backupWorkflow           *service.BackupService
	browseWorkflow           *service.BrowseWorkflow
	reportWorkflow           *service.ReportWorkflow
	labelWorkflow            *service.LabelWorkflow
	labelPresetWorkflow      *service.LabelPresetWorkflow
	relations                *service.RelationService
	dbMu                     sync.Mutex
	cfg                      Config
	authR                    *auth.Router
	settingsR                *settings.Router
	assetsR                  *assets.Router
	systemR                  *system.Router
}

type AuthClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	UserType int64  `json:"userType"`
	Source   string `json:"source"`
	jwt.RegisteredClaims
}

// resolveJWTSecret 返回生效的 JWT 签名密钥：显式配置优先；未配置时生成随机密钥
// 并持久化到数据库，使已签发会话的生命周期跟随数据库——删除数据库重建后旧令牌全部失效。
func resolveJWTSecret(db *sql.DB, configured string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		return configured, nil
	}
	var secret string
	err := db.QueryRow("SELECT jwt_secret FROM system_secrets WHERE id=1").Scan(&secret)
	if err == nil && strings.TrimSpace(secret) != "" {
		return secret, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret = hex.EncodeToString(raw)
	now := time.Now().Unix()
	if _, err := db.Exec("INSERT INTO system_secrets(id,jwt_secret,updated_at) VALUES(1,?,?) ON CONFLICT(id) DO UPDATE SET jwt_secret=excluded.jwt_secret,updated_at=excluded.updated_at", secret, now); err != nil {
		return "", err
	}
	return secret, nil
}

func (a *App) setDB(db *sql.DB) {
	a.db = db
	store := repository.NewStore(db)
	a.sql = service.NewSQLService(store)
	a.domains = service.NewDomainServices(store)
	a.files = service.NewFileStorageService(store, a.cfg.UploadDir)
	a.relations = service.NewRelationService(a.cfg.UploadDir)
	a.queries = repository.NewSQL(db)
	a.audit = service.NewAuditService(db, a.cfg.HistoryLimit)
	a.tagWorkflow = service.NewTagWorkflow(store, a.audit)
	a.userWorkflow = service.NewUserWorkflow(repository.NewUserRepository(store), a.audit)
	a.agentWorkflow = service.NewAgentWorkflow(repository.NewAgentRepository(store), a.audit)
	a.itemWorkflow = service.NewItemWorkflow(repository.NewItemRepository(store))
	a.itemActionWorkflow = service.NewItemActionWorkflow(repository.NewItemRepository(store), a.audit)
	a.itemMutationWorkflow = service.NewItemMutationWorkflow(repository.NewItemRepository(store), a.relations, a.audit)
	a.itemTagWorkflow = service.NewItemTagWorkflow(store, a.tagWorkflow, a.audit)
	a.softwareWorkflow = service.NewSoftwareWorkflow(repository.NewSoftwareRepository(store))
	a.softwareMutationWorkflow = service.NewSoftwareMutationWorkflow(repository.NewSoftwareRepository(store), a.relations, a.audit)
	a.softwareTagWorkflow = service.NewSoftwareTagWorkflow(store, a.tagWorkflow, a.audit)
	a.invoiceWorkflow = service.NewInvoiceWorkflow(repository.NewInvoiceRepository(store), a.relations, a.audit)
	a.contractWorkflow = service.NewContractWorkflow(repository.NewContractRepository(store), a.relations, a.audit)
	a.contractEventWorkflow = service.NewContractEventWorkflow(repository.NewContractRepository(store), a.audit)
	a.rackWorkflow = service.NewRackWorkflow(repository.NewInfrastructureRepository(store), a.audit)
	a.locationWorkflow = service.NewLocationWorkflow(repository.NewInfrastructureRepository(store), a.audit)
	a.backupWorkflow = service.NewBackupService(store, a.cfg.DBPath)
	a.detailRelations = service.NewDetailRelationsWorkflow(store)
	a.reportWorkflow = service.NewReportWorkflow(store)
	a.browseWorkflow = service.NewBrowseWorkflow(repository.NewToolRepository(store))
	a.labelWorkflow = service.NewLabelWorkflow(repository.NewToolRepository(store))
	a.labelPresetWorkflow = service.NewLabelPresetWorkflow(repository.NewToolRepository(store), a.audit)
	a.authWorkflow = service.NewAuthWorkflow(repository.NewAuthRepository(store), a.cfg.JWTSecret, func(ctx context.Context, username, password string) error {
		err := settings.AuthenticateLDAPUser(a.db, a.cfg.JWTSecret, username, password)
		if settings.IsLDAPInvalidCredentialsError(err) {
			return service.ErrInvalidCredentials
		}
		return err
	}, a.cfg.SessionTTL)
}

// assetsDeps 汇总资产域路由依赖
func (a *App) assetsDeps() assets.Deps {
	return assets.Deps{
		DB: a.db, SQL: a.sql, Queries: a.queries, Domains: a.domains, Audit: a.audit,
		Files: a.files, Relations: a.relations, TagWorkflow: a.tagWorkflow, AgentWorkflow: a.agentWorkflow,
		ItemWorkflow: a.itemWorkflow, ItemActionWorkflow: a.itemActionWorkflow,
		ItemMutationWorkflow: a.itemMutationWorkflow, ItemTagWorkflow: a.itemTagWorkflow,
		SoftwareWorkflow: a.softwareWorkflow, SoftwareMutationWorkflow: a.softwareMutationWorkflow,
		SoftwareTagWorkflow: a.softwareTagWorkflow, InvoiceWorkflow: a.invoiceWorkflow,
		ContractWorkflow: a.contractWorkflow, ContractEventWorkflow: a.contractEventWorkflow,
		RackWorkflow: a.rackWorkflow, LocationWorkflow: a.locationWorkflow,
		DetailRelations: a.detailRelations, BrowseWorkflow: a.browseWorkflow, ReportWorkflow: a.reportWorkflow,
		LabelWorkflow: a.labelWorkflow, LabelPresetWorkflow: a.labelPresetWorkflow,
		CurrentUserPermissions: a.currentUserPermissions, Cfg: a.cfg,
	}
}

// onDatabaseReplaced 数据库导入后重建服务依赖并原地刷新各域路由，保持已建立的处理器链可用
func (a *App) onDatabaseReplaced(newDB *sql.DB) {
	a.setDB(newDB)
	a.backupWorkflow = service.NewBackupService(repository.NewStore(newDB), a.cfg.DBPath)
	*a.authR = *auth.New(a.db, a.domains, a.audit, a.authWorkflow, a.cfg)
	*a.settingsR = *settings.New(a.db, a.sql, a.queries, a.audit, a.userWorkflow, a.cfg)
	*a.assetsR = *assets.New(a.assetsDeps())
	a.systemR.Reset(a.db, a.sql, a.queries, a.domains, a.audit, a.backupWorkflow, a.cfg)
}

// Run 使用环境变量与 .env 文件加载的默认配置启动后端服务
func Run() {
	RunWithConfig(config.Load())
}

// RunWithConfig 使用指定配置启动后端服务，供命令行参数覆盖场景复用完整启动流程
func RunWithConfig(cfg config.Config) {
	if err := system.EnsureDatabaseInitialized(cfg); err != nil {
		log.Fatalf("Database initialization failed: %s", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalf("Open database failed: %s", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := common.SetupSQLite(db); err != nil {
		log.Fatalf("Initialize SQLite settings failed: %s", err)
	}

	if err := system.EnsureRuntimeSchema(db, cfg.DBPath); err != nil {
		log.Fatalf("Upgrade database schema failed: %s", err)
	}

	jwtSecret, err := resolveJWTSecret(db, cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Resolve JWT secret failed: %s", err)
	}
	cfg.JWTSecret = jwtSecret
	if err := common.EnsureSettingsSecretsEncrypted(db, cfg.JWTSecret); err != nil {
		log.Fatalf("Encrypt settings secrets failed: %s", err)
	}

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("Create upload directory failed: %s", err)
	}

	app := &App{cfg: cfg}
	app.setDB(db)
	router := app.routes()

	go app.systemR.StartBackupScheduler()

	addr := cfg.ServerAddr
	log.Printf("ITDB Go API %s started, listening on %s", buildinfo.Version, addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}

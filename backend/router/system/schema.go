// 数据库运行时结构保障链：服务启动与数据库导入共用同一顺序的结构升级与默认数据函数，保证两种路径得到一致的库结构。
package system

import (
	"database/sql"

	"itdb-backend/router/common"
	"itdb-backend/router/settings"
)

// EnsureRuntimeSchema 按启动顺序执行全部结构与默认数据保障，导入数据库后同样调用
func EnsureRuntimeSchema(db *sql.DB, dbPath string) error {
	if err := common.EnsureStatusTypeColorSchema(db, dbPath); err != nil {
		return err
	}
	if err := common.EnsureLabelPapersSchema(db, dbPath); err != nil {
		return err
	}
	if err := common.EnsureActionsSchema(db, dbPath); err != nil {
		return err
	}
	if err := common.EnsureItemTypeSoftwareDefaults(db); err != nil {
		return err
	}
	if err := common.EnsureHistoryAuditSchema(db); err != nil {
		return err
	}
	if err := common.EnsureTagLinkOrphanCleanup(db); err != nil {
		return err
	}
	if err := common.EnsureSettingsSchema(db, dbPath); err != nil {
		return err
	}
	if err := common.EnsurePasswordResetSchema(db, dbPath); err != nil {
		return err
	}
	if err := settings.EnsureResourceSchema(db); err != nil {
		return err
	}
	return settings.EnsureUserProfileLoginSchema(db, dbPath)
}

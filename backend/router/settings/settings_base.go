package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"itdb-backend/router/common"
	"net/http"
	"time"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/service"
)

type publicBrandingResponse struct {
	SiteName    string `json:"siteName"`
	LoginName   string `json:"loginName"`
	AppName     string `json:"appName"`
	AppSubtitle string `json:"appSubtitle"`
	IconData    string `json:"iconData"`
}

func (a *Router) handleSettingsBase(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		common.WriteJSON(w, http.StatusOK, LoadSystemBaseConfig(r.Context(), a.db))
		return
	}
	var body domain.SystemBaseConfig
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	existing := LoadSystemBaseConfig(r.Context(), a.db)
	merged := body
	if body.Section != "" {
		result, ok := domain.MergeSystemBaseConfigSection(existing, body, body.Section)
		if !ok {
			common.WriteError(w, http.StatusBadRequest, "无效的基础配置区块")
			return
		}
		merged = result
	}
	config := domain.NormalizeSystemBaseConfig(merged)
	if err := domain.ValidateSystemBaseConfig(config); err != nil {
		common.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	changed := config != existing
	raw, err := json.Marshal(config)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := a.db.ExecContext(r.Context(), "UPDATE settings_base SET config=?,updated_at=? WHERE id=1", string(raw), time.Now().Unix()); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if changed {
		if actor, actorErr := common.CurrentUser(r.Context()); actorErr == nil {
			target, detail := baseSectionAuditText(body.Section)
			a.recordAuditEvent(r.Context(), actor.Username, common.ClientIP(r), service.AuditModuleSettings, "更新基础配置", target, detail, service.AuditResultSuccess)
		}
	}
	common.WriteJSON(w, http.StatusOK, config)
}

// baseSectionAuditText 基础配置审计按保存区块细分目标与详情
func baseSectionAuditText(section string) (string, string) {
	switch section {
	case domain.BaseSectionBrand:
		return "品牌标识", "品牌标识配置已更新"
	case domain.BaseSectionSecurity:
		return "安全时效", "安全时效配置已更新"
	case domain.BaseSectionBackup:
		return "数据备份", "数据备份配置已更新"
	}
	return "基础配置", "基础配置已更新"
}

// handlePublicBase 返回免认证的品牌标识，供登录页、找回密码页与启动屏动态展示。
func (a *Router) handlePublicBase(w http.ResponseWriter, r *http.Request) {
	config := LoadSystemBaseConfig(r.Context(), a.db)
	common.WriteJSON(w, http.StatusOK, publicBrandingResponse{
		SiteName:    config.SiteName,
		LoginName:   config.LoginName,
		AppName:     config.AppName,
		AppSubtitle: config.AppSubtitle,
		IconData:    config.IconData,
	})
}

// loadSystemBaseConfig 读取基础配置，缺行或解析失败时回退默认值。
func LoadSystemBaseConfig(ctx context.Context, db *sql.DB) domain.SystemBaseConfig {
	var raw string
	if err := db.QueryRowContext(ctx, "SELECT config FROM settings_base WHERE id=1").Scan(&raw); err != nil {
		return domain.DefaultSystemBaseConfig()
	}
	var config domain.SystemBaseConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return domain.DefaultSystemBaseConfig()
	}
	return domain.NormalizeSystemBaseConfig(config)
}

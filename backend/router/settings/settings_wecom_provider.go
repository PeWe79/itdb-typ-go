package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
)

// handleSettingsWecomProvider 企业微信认证配置读取与保存：Secret 加密存储且不回显
func (a *Router) handleSettingsWecomProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		provider, err := loadWecomProviderRow(r.Context(), a.db)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		responseConfig := map[string]any{
			"corpid":         provider.corpID,
			"agentid":        provider.agentID,
			"redirectPrefix": provider.redirectPrefix,
			"hasSecret":      provider.secret != "",
		}
		common.WriteJSON(w, http.StatusOK, map[string]any{
			"id": service.WecomProviderID, "type": service.WecomProviderID,
			"name": provider.name, "enabled": provider.enabled,
			"config": responseConfig, "updatedAt": formatUnixTime(provider.updatedAt),
		})
		return
	}
	var body struct {
		Name        string
		Enabled     bool
		ClearConfig bool
		Config      map[string]any
	}
	if !common.DecodeJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		common.WriteError(w, 400, "显示名称不能为空")
		return
	}
	if body.Config == nil {
		body.Config = map[string]any{}
	}
	for key, value := range body.Config {
		if text, ok := value.(string); ok {
			body.Config[key] = strings.TrimSpace(text)
		}
	}
	corpID := stringConfig(body.Config, "corpid")
	agentID := stringConfig(body.Config, "agentid")
	redirectPrefix := strings.TrimRight(stringConfig(body.Config, "redirectPrefix"), "/")
	secret := stringConfig(body.Config, "secret")

	existing, err := loadWecomProviderRow(r.Context(), a.db)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if body.ClearConfig {
		secret = ""
	} else if secret == "" {
		secret = existing.secret
	}
	if secret != "" && !strings.HasPrefix(secret, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(secret)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		secret = encrypted
	}
	if body.Enabled {
		if corpID == "" {
			common.WriteError(w, 400, "企业 ID（corpid）不能为空")
			return
		}
		if agentID == "" {
			common.WriteError(w, 400, "应用 AgentID 不能为空")
			return
		}
		if secret == "" {
			common.WriteError(w, 400, "应用 Secret 不能为空")
			return
		}
	}
	storedConfig := map[string]any{"corpid": corpID, "agentid": agentID, "redirectPrefix": redirectPrefix}
	if secret != "" {
		storedConfig["secret"] = secret
	}
	rawConfig, err := json.Marshal(storedConfig)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := a.db.ExecContext(r.Context(),
		"UPDATE settings_auth_providers SET enabled=?,name=?,config=?,updated_at=? WHERE id=?",
		boolInt(body.Enabled), name, string(rawConfig), time.Now().Unix(), service.WecomProviderID,
	); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	changed := boolInt(body.Enabled) != boolInt(existing.enabled) ||
		name != existing.name ||
		secret != existing.secret ||
		!reflect.DeepEqual(storedConfigWithoutSecret(storedConfig), storedConfigWithoutSecret(existing.configMap))
	if changed {
		if operator, operatorErr := common.CurrentUser(r.Context()); operatorErr == nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "更新认证配置", name, name+" 认证配置已更新", service.AuditResultSuccess)
		}
	}
	responseConfig := map[string]any{
		"corpid":         corpID,
		"agentid":        agentID,
		"redirectPrefix": redirectPrefix,
		"hasSecret":      secret != "",
	}
	common.WriteJSON(w, 200, map[string]any{
		"id": service.WecomProviderID, "type": service.WecomProviderID,
		"name": name, "enabled": body.Enabled,
		"config": responseConfig, "updatedAt": formatUnixTime(time.Now().Unix()),
	})
}

// wecomProviderRow settings_auth_providers 中企业微信配置的现状快照
type wecomProviderRow struct {
	name           string
	enabled        bool
	corpID         string
	agentID        string
	redirectPrefix string
	secret         string
	updatedAt      int64
	configMap      map[string]any
}

// loadWecomProviderRow 读取企业微信配置行，缺行时返回零值快照
func loadWecomProviderRow(ctx context.Context, db *sql.DB) (wecomProviderRow, error) {
	row := wecomProviderRow{configMap: map[string]any{}}
	var name, rawConfig string
	var enabled, updatedAt int64
	err := db.QueryRowContext(ctx,
		"SELECT name,enabled,COALESCE(config,''),updated_at FROM settings_auth_providers WHERE id=?",
		service.WecomProviderID,
	).Scan(&name, &enabled, &rawConfig, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return row, nil
	}
	if err != nil {
		return row, err
	}
	row.name = name
	row.enabled = enabled != 0
	row.updatedAt = updatedAt
	_ = json.Unmarshal([]byte(rawConfig), &row.configMap)
	row.corpID = wecomRowString(row.configMap, "corpid")
	row.agentID = wecomRowString(row.configMap, "agentid")
	row.redirectPrefix = wecomRowString(row.configMap, "redirectPrefix")
	row.secret = wecomRowString(row.configMap, "secret")
	return row, nil
}

func wecomRowString(stored map[string]any, key string) string {
	value, _ := stored[key].(string)
	return strings.TrimSpace(value)
}

// storedConfigWithoutSecret 复制配置并剔除加密 Secret，用于变更比对与响应
func storedConfigWithoutSecret(source map[string]any) map[string]any {
	clone := map[string]any{}
	for key, value := range source {
		if key == "secret" {
			continue
		}
		clone[key] = value
	}
	return clone
}

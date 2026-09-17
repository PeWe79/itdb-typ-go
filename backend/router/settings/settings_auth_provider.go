package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"itdb-backend/internal/security"
	"itdb-backend/internal/service"
	"itdb-backend/router/common"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func (a *Router) handleSettingsAuthProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var enabled int64
		var server, dn, bindDN, getUsers, filter, bindPassword, providerName, providerConfig string
		if err := a.db.QueryRowContext(r.Context(), "SELECT s.useldap,s.ldap_server,s.ldap_dn,s.ldap_bind_dn,s.ldap_getusers,s.ldap_getusers_filter,s.ldap_bind_password,COALESCE(NULLIF(p.name,''),'AD/LDAP'),COALESCE(p.config,'') FROM settings s LEFT JOIN settings_auth_providers p ON p.id='ldap' LIMIT 1").Scan(&enabled, &server, &dn, &bindDN, &getUsers, &filter, &bindPassword, &providerName, &providerConfig); err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		config := map[string]any{"host": server, "baseDN": dn, "bindDN": bindDN, "userFilter": getUsers, "groupFilter": filter, "hasBindPassword": strings.TrimSpace(bindPassword) != ""}
		var stored map[string]any
		if err := json.Unmarshal([]byte(providerConfig), &stored); err == nil {
			for key, value := range stored {
				if key == "hasBindPassword" {
					continue
				}
				config[key] = value
			}
		}
		common.WriteJSON(w, 200, map[string]any{"id": "ldap", "type": "ldap", "name": providerName, "enabled": enabled == 1, "config": config, "updatedAt": a.ldapProviderUpdatedAt(r.Context())})
		return
	}
	if r.Method == http.MethodPost {
		operator, _ := common.CurrentUser(r.Context())
		providerName := a.ldapProviderDisplayName(r.Context())
		cfg, err := loadLDAPSettings(a.db, a.cfg.JWTSecret)
		if err != nil {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		conn, err := dialAndBindLDAP(cfg)
		if err != nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, ldapUserMessage(err), service.AuditResultFailure)
			common.WriteError(w, http.StatusServiceUnavailable, ldapUserMessage(err))
			return
		}
		matchedUsers, countErr := countLDAPUsers(conn, cfg)
		_ = conn.Close()
		if countErr != nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, ldapUserMessage(countErr), service.AuditResultFailure)
			common.WriteError(w, http.StatusServiceUnavailable, ldapUserMessage(countErr))
			return
		}
		a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "测试认证配置", providerName, fmt.Sprintf("AD/LDAP 认证配置测试成功，匹配用户数：%d", matchedUsers), service.AuditResultSuccess)
		common.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "matchedUsers": matchedUsers})
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
	delete(body.Config, "hasBindPassword")
	host := stringConfig(body.Config, "host")
	baseDN := stringConfig(body.Config, "baseDN")
	bindDN := stringConfig(body.Config, "bindDN")
	filter := stringConfig(body.Config, "userFilter")
	var port int
	switch value := body.Config["port"].(type) {
	case float64:
		port = int(value)
	case string:
		port, _ = strconv.Atoi(value)
	}
	var existingPassword string
	var existingSettings settingsLDAPRow
	if err := a.db.QueryRowContext(r.Context(), "SELECT useldap,COALESCE(ldap_server,''),COALESCE(ldap_dn,''),COALESCE(ldap_bind_dn,''),COALESCE(ldap_bind_password,''),COALESCE(ldap_getusers,''),COALESCE(ldap_getusers_filter,'') FROM settings LIMIT 1").Scan(
		&existingSettings.UseLDAP, &existingSettings.Server, &existingSettings.BaseDN,
		&existingSettings.BindDN, &existingSettings.BindPassword, &existingSettings.GetUsers, &existingSettings.Filter,
	); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	existingPassword = existingSettings.BindPassword
	var existingProviderEnabled int64
	var existingProviderName, existingProviderRaw string
	if err := a.db.QueryRowContext(r.Context(), "SELECT enabled,name,COALESCE(config,'{}') FROM settings_auth_providers WHERE id='ldap'").Scan(&existingProviderEnabled, &existingProviderName, &existingProviderRaw); err != nil && !errors.Is(err, sql.ErrNoRows) {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	password := stringConfig(body.Config, "bindPassword")
	if body.ClearConfig {
		password = ""
	} else if password == "" {
		password = existingPassword
	}
	if password != "" && !strings.HasPrefix(password, "enc:v1:") {
		encrypted, err := security.EncryptSettingsSecret(password)
		if err != nil {
			common.WriteError(w, 500, err.Error())
			return
		}
		password = encrypted
	}
	if body.Enabled {
		if host == "" {
			common.WriteError(w, 400, "LDAP 服务器地址不能为空")
			return
		}
		if baseDN == "" {
			common.WriteError(w, 400, "Base DN 不能为空")
			return
		}
		if filter == "" {
			common.WriteError(w, 400, "用户过滤器不能为空")
			return
		}
		if bindDN == "" {
			common.WriteError(w, 400, "绑定 DN 不能为空")
			return
		}
		if password == "" {
			common.WriteError(w, 400, "绑定密码不能为空")
			return
		}
		if settingsBool(body.Config, "useTLS") && settingsBool(body.Config, "startTLS") {
			common.WriteError(w, 400, "LDAPS 与 StartTLS 不能同时启用")
			return
		}
		if port <= 0 {
			common.WriteError(w, 400, "端口不能为空")
			return
		}
	}
	providerConfig := cloneMap(body.Config)
	delete(providerConfig, "bindPassword")
	if password != "" {
		providerConfig["hasBindPassword"] = true
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(r.Context(), "UPDATE settings SET useldap=?,ldap_server=?,ldap_dn=?,ldap_bind_dn=?,ldap_bind_password=?,ldap_getusers=?,ldap_getusers_filter=?", boolInt(body.Enabled), host, baseDN, bindDN, password, filter, stringConfig(body.Config, "groupFilter")); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	if _, err = tx.ExecContext(r.Context(), "UPDATE settings_auth_providers SET enabled=?,name=?,config=?,updated_at=? WHERE id='ldap'", boolInt(body.Enabled), name, mustJSON(providerConfig), time.Now().Unix()); err != nil {
		common.WriteError(w, 500, err.Error())
		return
	}
	if err = tx.Commit(); err != nil {
		common.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	authChanged := boolInt(body.Enabled) != existingSettings.UseLDAP ||
		host != existingSettings.Server ||
		baseDN != existingSettings.BaseDN ||
		bindDN != existingSettings.BindDN ||
		password != existingSettings.BindPassword ||
		filter != existingSettings.GetUsers ||
		stringConfig(body.Config, "groupFilter") != existingSettings.Filter ||
		boolInt(body.Enabled) != existingProviderEnabled ||
		name != existingProviderName ||
		!reflect.DeepEqual(providerConfig, decodeSettingsMap(existingProviderRaw))
	if authChanged {
		if operator, operatorErr := common.CurrentUser(r.Context()); operatorErr == nil {
			a.recordAuditEvent(r.Context(), operator.Username, common.ClientIP(r), service.AuditModuleSettings, "更新认证配置", existingProviderName, existingProviderName+" 认证配置已更新", service.AuditResultSuccess)
		}
	}
	responseConfig := cloneMap(body.Config)
	delete(responseConfig, "bindPassword")
	if password != "" {
		responseConfig["hasBindPassword"] = true
	}
	common.WriteJSON(w, 200, map[string]any{"id": "ldap", "type": "ldap", "name": name, "enabled": body.Enabled, "config": responseConfig, "updatedAt": a.ldapProviderUpdatedAt(r.Context())})
}

// settingsLDAPRow settings 表认证配置的现状快照
type settingsLDAPRow struct {
	UseLDAP      int64
	Server       string
	BaseDN       string
	BindDN       string
	BindPassword string
	GetUsers     string
	Filter       string
}

func (a *Router) ldapProviderUpdatedAt(ctx context.Context) string {
	var updatedAt int64
	if err := a.db.QueryRowContext(ctx, "SELECT updated_at FROM settings_auth_providers WHERE id='ldap'").Scan(&updatedAt); err != nil || updatedAt <= 0 {
		return ""
	}
	return formatUnixTime(updatedAt)
}

func (a *Router) ldapProviderDisplayName(ctx context.Context) string {
	var name string
	if err := a.db.QueryRowContext(ctx, "SELECT COALESCE(NULLIF(name,''),'AD/LDAP') FROM settings_auth_providers WHERE id='ldap'").Scan(&name); err != nil {
		return "AD/LDAP"
	}
	return name
}

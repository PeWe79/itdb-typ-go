package domain

import (
	"errors"
	"math"
	"strings"
	"unicode/utf8"
)

// SystemBaseConfig 系统基础配置：品牌标识、安全时效与数据备份计划。
type SystemBaseConfig struct {
	SiteName                         string  `json:"siteName" example:"ITDB"`
	LoginName                        string  `json:"loginName" example:"ITDB"`
	AppName                          string  `json:"appName" example:"ITDB"`
	AppSubtitle                      string  `json:"appSubtitle" example:"IT Asset Management"`
	IconData                         string  `json:"iconData" example:"/favicon.svg"`
	ResetCodeTTLMinutes              int     `json:"resetCodeTtlMinutes" example:"10"`
	ResetCaptchaTTLMinutes           int     `json:"resetCaptchaTtlMinutes" example:"1"`
	PasswordResetSendCooldownMinutes float64 `json:"passwordResetSendCooldownMinutes" example:"0.5"`
	PasswordResetRateLimitMinutes    int     `json:"passwordResetRateLimitMinutes" example:"5"`
	WecomStateTTLMinutes             int     `json:"wecomStateTtlMinutes" example:"5"`
	BackupEnabled                    bool    `json:"backupEnabled"`
	BackupCron                       string  `json:"backupCron" example:"0 0 * * *"`
	BackupRetentionDays              int     `json:"backupRetentionDays" example:"30"`
	Section                          string  `json:"section,omitempty" example:"brand"`
}

// 基础配置的三个区块标识：brand 品牌标识、security 安全时效、backup 数据备份
const (
	BaseSectionBrand    = "brand"
	BaseSectionSecurity = "security"
	BaseSectionBackup   = "backup"
)

// DefaultSystemBaseConfig 返回系统基础配置默认值。
func DefaultSystemBaseConfig() SystemBaseConfig {
	return SystemBaseConfig{
		SiteName:                         "ITDB",
		LoginName:                        "ITDB",
		AppName:                          "ITDB",
		AppSubtitle:                      "IT Asset Management",
		IconData:                         "/favicon.svg",
		ResetCodeTTLMinutes:              10,
		ResetCaptchaTTLMinutes:           1,
		PasswordResetSendCooldownMinutes: 0.5,
		PasswordResetRateLimitMinutes:    5,
		WecomStateTTLMinutes:             5,
		BackupCron:                       "0 0 * * *",
		BackupRetentionDays:              30,
	}
}

// NormalizeSystemBaseConfig 为空值与非法数值回填默认值。
func NormalizeSystemBaseConfig(config SystemBaseConfig) SystemBaseConfig {
	def := DefaultSystemBaseConfig()
	config.SiteName = baseText(config.SiteName, def.SiteName)
	config.LoginName = baseText(config.LoginName, def.LoginName)
	config.AppName = baseText(config.AppName, def.AppName)
	config.AppSubtitle = baseText(config.AppSubtitle, def.AppSubtitle)
	config.IconData = baseText(config.IconData, def.IconData)
	config.ResetCodeTTLMinutes = baseInt(config.ResetCodeTTLMinutes, def.ResetCodeTTLMinutes)
	config.ResetCaptchaTTLMinutes = baseInt(config.ResetCaptchaTTLMinutes, def.ResetCaptchaTTLMinutes)
	if !(config.PasswordResetSendCooldownMinutes > 0) {
		config.PasswordResetSendCooldownMinutes = def.PasswordResetSendCooldownMinutes
	}
	config.PasswordResetRateLimitMinutes = baseInt(config.PasswordResetRateLimitMinutes, def.PasswordResetRateLimitMinutes)
	config.WecomStateTTLMinutes = baseInt(config.WecomStateTTLMinutes, def.WecomStateTTLMinutes)
	config.BackupCron = baseText(config.BackupCron, def.BackupCron)
	if config.BackupRetentionDays < 0 {
		config.BackupRetentionDays = def.BackupRetentionDays
	}
	return config
}

// ValidateSystemBaseConfig 校验系统基础配置取值范围。
func ValidateSystemBaseConfig(config SystemBaseConfig) error {
	for _, item := range []struct {
		label string
		value string
	}{
		{"网站名称", config.SiteName},
		{"认证页品牌名称", config.LoginName},
		{"控制台品牌名称", config.AppName},
		{"控制台品牌副标题", config.AppSubtitle},
	} {
		if utf8.RuneCountInString(item.value) > 60 {
			return errors.New(item.label + "长度不能超过 60 个字符")
		}
	}
	if !strings.HasPrefix(config.IconData, "/") && !strings.HasPrefix(config.IconData, "data:image/") {
		return errors.New("图标格式无效，须为站点路径或图片 Data URL")
	}
	if config.ResetCodeTTLMinutes < 1 || config.ResetCodeTTLMinutes > 60 {
		return errors.New("找回密码验证码有效期范围为 1-60 分钟")
	}
	if config.ResetCaptchaTTLMinutes < 1 || config.ResetCaptchaTTLMinutes > 10 {
		return errors.New("图形验证码有效期范围为 1-10 分钟")
	}
	if config.PasswordResetSendCooldownMinutes < 0.5 || config.PasswordResetSendCooldownMinutes > 10 || !isHalfMinuteStep(config.PasswordResetSendCooldownMinutes) {
		return errors.New("发送冷却时间范围为 0.5-10 分钟，且步长为 0.5 分钟")
	}
	if config.PasswordResetRateLimitMinutes < 5 || config.PasswordResetRateLimitMinutes > 10 {
		return errors.New("频率限制统计窗口范围为 5-10 分钟")
	}
	if config.WecomStateTTLMinutes < 1 || config.WecomStateTTLMinutes > 60 {
		return errors.New("企业微信扫码有效期范围为 1-60 分钟")
	}
	if len(strings.Fields(config.BackupCron)) != 5 {
		return errors.New("定时备份 Cron 需为五段表达式：分 时 日 月 周")
	}
	if config.BackupRetentionDays > 3650 {
		return errors.New("备份保留天数范围为 0-3650 天，0 表示永久保留")
	}
	return nil
}

func isHalfMinuteStep(value float64) bool {
	return math.Abs(value*2-math.Round(value*2)) < 1e-6
}

func baseText(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// MergeSystemBaseConfigSection 按区块白名单用 patch 覆盖 existing 的对应字段，
// 其余字段保留 existing 原值，避免整包提交相互覆盖；section 非法时第二个返回值为 false。
func MergeSystemBaseConfigSection(existing, patch SystemBaseConfig, section string) (SystemBaseConfig, bool) {
	switch section {
	case BaseSectionBrand:
		existing.SiteName = patch.SiteName
		existing.LoginName = patch.LoginName
		existing.AppName = patch.AppName
		existing.AppSubtitle = patch.AppSubtitle
		existing.IconData = patch.IconData
	case BaseSectionSecurity:
		existing.ResetCodeTTLMinutes = patch.ResetCodeTTLMinutes
		existing.ResetCaptchaTTLMinutes = patch.ResetCaptchaTTLMinutes
		existing.PasswordResetSendCooldownMinutes = patch.PasswordResetSendCooldownMinutes
		existing.PasswordResetRateLimitMinutes = patch.PasswordResetRateLimitMinutes
		existing.WecomStateTTLMinutes = patch.WecomStateTTLMinutes
	case BaseSectionBackup:
		existing.BackupEnabled = patch.BackupEnabled
		existing.BackupCron = patch.BackupCron
		existing.BackupRetentionDays = patch.BackupRetentionDays
	default:
		return existing, false
	}
	return existing, true
}

func baseInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

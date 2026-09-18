package router

type swaggerErrorResponse struct {
	// 错误信息
	Error string `json:"error" example:"请求参数错误"`
}

type swaggerIDResponse struct {
	// 编号
	ID int64 `json:"id" example:"1"`
}

type swaggerOKResponse struct {
	// 是否成功
	OK bool `json:"ok" example:"true"`
}

type swaggerHealthResponse struct {
	// 状态
	Status string `json:"status" example:"ok"`
}

type swaggerMessageResponse struct {
	// 消息
	Message string `json:"message" example:"操作成功"`
}

type swaggerImportDatabaseResponse struct {
	// 是否成功
	OK bool `json:"ok" example:"true"`
	// 消息
	Message string `json:"message" example:"数据库导入成功"`
}

type swaggerGenericObject map[string]interface{}

type swaggerDashboardSummaryResponse struct {
	// 各资源总数
	Counts map[string]int64 `json:"counts"`
}

type swaggerReportRunResponse struct {
	Meta  map[string]interface{}   `json:"meta"`
	Rows  []map[string]interface{} `json:"rows"`
	Chart []map[string]interface{} `json:"chart"`
}

type swaggerAuditHistoryItem struct {
	// 编号
	ID int64 `json:"id" example:"1"`
	// 时间戳（Unix 秒）
	Timestamp int64 `json:"timestamp" example:"1725868800"`
	// 用户名
	Username string `json:"username" example:"admin"`
	// 模块
	Module string `json:"module" example:"auth"`
	// 操作
	Action string `json:"action" example:"用户登录"`
	// 目标
	Target      string `json:"target" example:"硬件编号 1"`
	TargetTitle string `json:"targetTitle" example:"部门1、部门2"`
	// 来源 IP
	IP string `json:"ip" example:"10.18.88.135"`
	// 结果
	Result string `json:"result" example:"success"`
	// 详情
	Detail string `json:"detail" example:"登录成功（本地账号）"`
}

type swaggerAuditHistoryResponse struct {
	Items []swaggerAuditHistoryItem `json:"items"`
	// 总数
	Total int64 `json:"total" example:"78"`
	// 返回条数上限
	Limit int `json:"limit" example:"500"`
}

type swaggerHistoryClearRequest struct {
	// 待清空的审计日志编号清单
	IDs []int64 `json:"ids" example:"1,2,3"`
}

type swaggerHistoryClearResponse struct {
	// 实际删除的记录条数
	Deleted int64 `json:"deleted" example:"12"`
}

type swaggerLabelPreviewRequest struct {
	// 要打印标签的硬件编号清单
	ItemIDs    []int64 `json:"itemIds" example:"1,2,3"` // 硬件编号清单
	PresetName string  `json:"presetName,omitempty" example:"Avery6106"`
	// 标签预设名称
	Preset string `json:"preset" example:"A4-3x8"`
}

type swaggerDictionaryPayload map[string]interface{}

type swaggerTagPayload struct {
	// 名称
	Name string `json:"name" example:"生产"`
}

type swaggerSettingsRole struct {
	// 编号
	ID string `json:"id"`
	// 角色标识
	Key string `json:"key"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 权限标识集合
	Permissions []string `json:"permissions"`
	// 是否内置
	Builtin bool `json:"builtin"`
	// 是否禁用
	Disabled bool `json:"disabled"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}
type swaggerSettingsUser struct {
	// 编号
	ID string `json:"id"`
	// 用户名
	Username string `json:"username"`
	// 显示名称
	DisplayName string `json:"displayName"`
	// 邮箱
	Email string `json:"email"`
	// 主角色
	Role string `json:"role"`
	// 账号来源
	Source string `json:"source"`
	// 有效角色
	Roles []swaggerSettingsRole `json:"roles"`
	// 直接分配角色
	DirectRoles []swaggerSettingsRole `json:"directRoles"`
	// 有效角色
	EffectiveUserRoles []swaggerSettingsRole `json:"effectiveUserRoles"`
	// 权限标识集合
	Permissions []string `json:"permissions"`
	// 有效权限清单
	EffectiveUserPermissions []string `json:"effectiveUserPermissions"`
	// 是否禁用
	Disabled bool `json:"disabled"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 最近登录时间
	LastLoginAt string `json:"lastLoginAt"`
}
type swaggerSettingsUserListResponse struct {
	Items []swaggerSettingsUser `json:"items"`
	// 总数
	Total int `json:"total"`
}
type swaggerSettingsRoleListResponse struct {
	Items []swaggerSettingsRole `json:"items"`
	// 总数
	Total int `json:"total"`
}
type swaggerSettingsGroup struct {
	// 编号
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 是否禁用
	Disabled bool `json:"disabled"`
	// 群组成员
	Members []swaggerSettingsUser `json:"members"`
	// 有效角色
	Roles []swaggerSettingsRole `json:"roles"`
}
type swaggerSettingsGroupListResponse struct {
	Items []swaggerSettingsGroup `json:"items"`
	// 总数
	Total int `json:"total"`
}
type swaggerSettingsAuthProviderResponse struct {
	// 编号
	ID string `json:"id"`
	// 类型
	Type string `json:"type"`
	// 名称
	Name string `json:"name"`
	// 是否启用
	Enabled bool `json:"enabled"`
	// 配置内容
	Config map[string]interface{} `json:"config"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}
type swaggerSettingsEmailResponse struct {
	// 编号
	ID string `json:"id"`
	// 名称
	Name string `json:"name"`
	// 是否启用找回密码
	PasswordResetEnabled bool `json:"passwordResetEnabled"`
	// 配置内容
	Config map[string]interface{} `json:"config"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}
type swaggerSystemBaseConfig struct {
	// 网站名称
	SiteName string `json:"siteName" example:"ITDB"`
	// 登录页品牌名称
	LoginName string `json:"loginName" example:"ITDB"`
	// 控制台品牌名称
	AppName string `json:"appName" example:"ITDB"`
	// 控制台品牌副标题
	AppSubtitle string `json:"appSubtitle" example:"IT Asset Management"`
	// 图标（站点路径或 Data URL）
	IconData string `json:"iconData" example:"/favicon.svg"`
	// 找回密码验证码有效期（分钟）
	ResetCodeTTLMinutes int `json:"resetCodeTtlMinutes" example:"10"`
	// 图形验证码有效期（分钟）
	ResetCaptchaTTLMinutes int `json:"resetCaptchaTtlMinutes" example:"1"`
	// 验证码发送冷却（分钟）
	PasswordResetSendCooldownMinutes float64 `json:"passwordResetSendCooldownMinutes" example:"0.5"`
	// 发送限流统计窗口（分钟）
	PasswordResetRateLimitMinutes int `json:"passwordResetRateLimitMinutes" example:"5"`
	// 是否启用定时备份
	BackupEnabled bool `json:"backupEnabled"`
	// 定时备份 Cron 表达式
	BackupCron string `json:"backupCron" example:"0 0 * * *"`
	// 备份保留天数（0 为永久）
	BackupRetentionDays int    `json:"backupRetentionDays" example:"30"`
	Section             string `json:"section,omitempty" example:"brand"`
}

type swaggerPublicBrandingResponse struct {
	// 网站名称
	SiteName string `json:"siteName" example:"ITDB"`
	// 登录页品牌名称
	LoginName string `json:"loginName" example:"ITDB"`
	// 控制台品牌名称
	AppName string `json:"appName" example:"ITDB"`
	// 控制台品牌副标题
	AppSubtitle string `json:"appSubtitle" example:"IT Asset Management"`
	// 图标（站点路径或 Data URL）
	IconData string `json:"iconData" example:"/favicon.svg"`
}

type swaggerPasswordResetCaptchaResponse struct {
	Token string `json:"token"`
	// 图形验证码题目
	Question string `json:"question"`
	// 过期时间
	ExpiresAt string `json:"expiresAt"`
}
type swaggerPasswordResetVerifyRequest struct {
	// 用户名
	Username      string `json:"username"`
	CaptchaToken  string `json:"captchaToken"`
	CaptchaAnswer string `json:"captchaAnswer"`
}
type swaggerPasswordResetVerifyResponse struct {
	// 流程令牌
	VerificationToken string `json:"verificationToken"`
	// 可用的验证码接收渠道
	Channels []map[string]interface{} `json:"channels"`
}
type swaggerPasswordResetSendRequest struct {
	// 流程令牌
	VerificationToken string `json:"verificationToken"`
	// 接收渠道
	Channel string `json:"channel"`
	// 接收验证码的邮箱
	VerifyEmail string `json:"verifyEmail"`
}
type swaggerPasswordResetSendResponse struct {
	// 状态
	Status string `json:"status"`
	// 发送冷却剩余秒数
	CooldownSeconds int `json:"cooldownSeconds"`
}
type swaggerPasswordResetConfirmRequest struct {
	// 流程令牌
	VerificationToken string `json:"verificationToken"`
	// 邮箱验证码
	Code string `json:"code"`
	// 新密码
	NewPassword string `json:"newPassword"`
	// 确认新密码
	ConfirmPassword string `json:"confirmPassword"`
}

type swaggerPublicAuthProvider struct {
	// 编号
	ID string `json:"id"`
	// 类型
	Type string `json:"type"`
	// 名称
	Name string `json:"name"`
	// 是否启用
	Enabled bool `json:"enabled"`
}

type swaggerPublicAuthProvidersResponse struct {
	Items []swaggerPublicAuthProvider `json:"items"`
	// 总数
	Total int `json:"total"`
	// 是否启用找回密码
	PasswordResetEnabled bool `json:"passwordResetEnabled"`
}

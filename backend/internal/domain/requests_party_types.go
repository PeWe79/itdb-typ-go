package domain

type AgentContact struct {
	Name     string `json:"name"`
	Phones   string `json:"phones"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Comments string `json:"comments"`
}

type AgentURL struct {
	Description string `json:"description"`
	URL         string `json:"url"`
}

type AgentPayload struct {
	// 代理类型集合
	Types []int64 `json:"types"`
	// 类型位掩码
	Type *int64 `json:"type"`
	// 代理名称
	Title string `json:"title"`
	// 联系信息
	ContactInfo string `json:"contactInfo"`
	// 联系人清单
	Contacts []AgentContact `json:"contacts"`
	// URL 清单
	URLs []AgentURL `json:"urls"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
}

type UserPayload struct {
	// 用户名
	Username string `json:"username"`
	// 显示名称
	UserDesc string `json:"userDesc"`
	// 密码
	Password string `json:"password"`
	// 用户类型（0 管理员）
	UserType int64 `json:"userType"`
}

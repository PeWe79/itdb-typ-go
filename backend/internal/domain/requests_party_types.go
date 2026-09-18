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
	Types       []int64        `json:"types"`
	Type        *int64         `json:"type"`
	Title       string         `json:"title"`
	ContactInfo string         `json:"contactInfo"`
	Contacts    []AgentContact `json:"contacts"`
	URLs        []AgentURL     `json:"urls"`
	ChangeNote  string         `json:"changeNote,omitempty"`
}

type UserPayload struct {
	Username string `json:"username"`
	UserDesc string `json:"userDesc"`
	Password string `json:"password"`
	UserType int64  `json:"userType"`
}

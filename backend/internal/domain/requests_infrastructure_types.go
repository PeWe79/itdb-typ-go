package domain

type LocationPayload struct {
	Name       string   `json:"name"`
	Floor      string   `json:"floor"`
	ChangeNote string   `json:"changeNote,omitempty"`
	Areas      []string `json:"areas,omitempty"`
}

type RackPayload struct {
	LocationID int64  `json:"locationId"`
	LocAreaID  *int64 `json:"locAreaId"`
	USize      int64  `json:"uSize"`
	RevNums    int64  `json:"revNums"`
	Depth      int64  `json:"depth"`
	Comments   string `json:"comments"`
	Model      string `json:"model"`
	Label      string `json:"label"`
	ChangeNote string `json:"changeNote,omitempty"`
}

type LocAreaPayload struct {
	AreaName string `json:"areaName"`
}
